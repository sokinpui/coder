package coagent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	coagentprompt "github.com/sokinpui/coder/internal/engine/coagent/prompt"
	"github.com/sokinpui/coder/internal/types"
)

type AgentRuntime struct {
	Config            config.Generation
	BaseURL           string
	APIKey            string
	MaxToolIterations int
	Registry          *Registry
}

func NewAgentRuntime(cfg *config.Config, registry *Registry) *AgentRuntime {
	if registry == nil {
		registry = DefaultRegistry
	}
	return &AgentRuntime{
		Config:            cfg.Generation,
		BaseURL:           cfg.Server.URL,
		APIKey:            cfg.Server.APIKey,
		MaxToolIterations: 20,
		Registry:          registry,
	}
}

func (ar *AgentRuntime) RunLoop(ctx context.Context, systemInstruction string, messages []types.ChatMessage, streamChan chan<- AgentStreamChunk) {
	defer close(streamChan)

	currentMessages := append([]types.ChatMessage(nil), messages...)
	maxIterations := ar.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 20
	}

	for range maxIterations {
		if ctx.Err() != nil {
			return
		}

		toolCalls, turnText, hasError := ar.executeTurn(ctx, systemInstruction, currentMessages, streamChan, true)
		if turnText != "" {
			currentMessages = append(currentMessages, types.ChatMessage{Role: types.RoleAssistant, Content: turnText})
		}
		if hasError || ctx.Err() != nil || len(toolCalls) == 0 {
			return
		}

		for _, tc := range toolCalls {
			if ctx.Err() != nil {
				return
			}

			streamChan <- AgentStreamChunk{ToolCall: &tc}
			output, err := ar.Registry.Execute(ctx, tc.Name, tc.Arguments)
			if err != nil {
				output = fmt.Sprintf("Error: %v", err)
			}

			resInfo := ToolResultInfo{
				CallID: tc.CallID,
				Name:   tc.Name,
				Output: output,
			}
			streamChan <- AgentStreamChunk{ToolResult: &resInfo}
		}
	}

	if ctx.Err() != nil {
		return
	}

	_, _, _ = ar.executeTurn(ctx, systemInstruction, currentMessages, streamChan, false)
}

func (ar *AgentRuntime) executeTurn(ctx context.Context, systemInstruction string, messages []types.ChatMessage, streamChan chan<- AgentStreamChunk, enableTools bool) ([]ToolCallInfo, string, bool) {
	var inputItems []any

	for _, msg := range messages {
		switch msg.Role {
		case types.RoleUser:
			if len(msg.Data) > 0 {
				b64 := base64.StdEncoding.EncodeToString(msg.Data)
				mimeType := "image/png"
				if len(msg.Data) > 4 && bytes.Equal(msg.Data[:4], []byte{0xFF, 0xD8, 0xFF, 0xE0}) {
					mimeType = "image/jpeg"
				}
				inputItems = append(inputItems, map[string]any{
					"type": "message",
					"role": "user",
					"content": []openAIContentPart{
						{
							Type: "input_image",
							ImageURL: &openAIImageURL{
								URL: fmt.Sprintf("data:%s;base64,%s", mimeType, b64),
							},
						},
					},
				})
				continue
			}
			if msg.Content != "" {
				inputItems = append(inputItems, map[string]any{
					"type":    "message",
					"role":    "user",
					"content": msg.Content,
				})
			}
		case types.RoleAssistant:
			if msg.Content != "" {
				inputItems = append(inputItems, map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": msg.Content,
				})
			}
		}
	}

	if !enableTools {
		inputItems = append(inputItems, map[string]any{
			"type":    "message",
			"role":    "user",
			"content": "Tool execution iteration limit reached. Do not call any tools. Provide your final response to the user based on the tool results so far.",
		})
	}

	instructions := systemInstruction
	if instructions == "" {
		instructions = coagentprompt.Instructions
	}

	body := map[string]any{
		"model":        ar.Config.ModelCode,
		"stream":       true,
		"store":        false,
		"instructions": instructions,
		"input":        inputItems,
	}

	toolDecls := ar.Registry.Declarations()
	if len(toolDecls) > 0 {
		body["tools"] = toolDecls
		if enableTools {
			body["tool_choice"] = "auto"
		} else {
			body["tool_choice"] = "none"
		}
	}

	if ar.Config.ReasoningEffort != "" {
		body["reasoning"] = map[string]any{
			"effort": ar.Config.ReasoningEffort,
		}
	}

	jsonBody, _ := json.Marshal(body)
	url := strings.TrimSuffix(ar.BaseURL, "/") + "/responses"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Failed to create request: %v", err)}
		return nil, "", true
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if ar.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+ar.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, "", false
		}
		streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Failed to connect to server: %v", err)}
		return nil, "", true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Server returned status %d: %s", resp.StatusCode, string(errBytes))}
		return nil, "", true
	}

	var toolCalls []ToolCallInfo
	var currentToolCall *ToolCallInfo
	var textBuilder strings.Builder

	reader := bufio.NewReader(resp.Body)
	for {
		if ctx.Err() != nil {
			return nil, textBuilder.String(), false
		}

		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if after, ok := strings.CutPrefix(line, "data:"); ok {
				raw := after
				if strings.HasPrefix(raw, " ") {
					raw = raw[1:]
				}
				trimmed := strings.TrimSpace(raw)
				if trimmed == "[DONE]" {
					break
				}

				var ev responseStreamEvent
				if err := json.Unmarshal([]byte(trimmed), &ev); err == nil {
					switch ev.Type {
					case "response.output_text.delta":
						if ev.Delta != "" {
							textBuilder.WriteString(ev.Delta)
							select {
							case <-ctx.Done():
								return nil, textBuilder.String(), false
							case streamChan <- AgentStreamChunk{Content: ev.Delta}:
							}
						}
					case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
						if ev.Delta != "" {
							select {
							case <-ctx.Done():
								return nil, textBuilder.String(), false
							case streamChan <- AgentStreamChunk{ReasoningContent: ev.Delta}:
							}
						}
					case "response.output_item.added":
						if ev.Item != nil && ev.Item.Type == "function_call" {
							currentToolCall = &ToolCallInfo{
								CallID:    ev.Item.CallID,
								Name:      ev.Item.Name,
								Arguments: ev.Item.Arguments,
							}
							if currentToolCall.CallID == "" {
								currentToolCall.CallID = ev.Item.ID
							}
						}
					case "response.function_call_arguments.delta":
						if currentToolCall != nil {
							currentToolCall.Arguments += ev.Delta
						}
					case "response.function_call_arguments.done":
						if currentToolCall != nil && ev.Arguments != "" {
							currentToolCall.Arguments = ev.Arguments
						}
					case "response.output_item.done":
						if ev.Item != nil && ev.Item.Type == "function_call" {
							cid := ev.Item.CallID
							if cid == "" {
								cid = ev.Item.ID
							}
							args := ev.Item.Arguments
							if args == "" && currentToolCall != nil {
								args = currentToolCall.Arguments
							}
							toolCalls = append(toolCalls, ToolCallInfo{
								CallID:    cid,
								Name:      ev.Item.Name,
								Arguments: args,
							})
							currentToolCall = nil
						}
					case "response.completed":
						return toolCalls, textBuilder.String(), false
					case "response.incomplete":
						return toolCalls, textBuilder.String(), false
					case "error", "response.failed":
						errMsg := trimmed
						if ev.Error != nil && ev.Error.Message != "" {
							errMsg = ev.Error.Message
						}
						streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: %s", errMsg)}
						return nil, textBuilder.String(), true
					}
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF && ctx.Err() == nil {
				streamChan <- AgentStreamChunk{Content: fmt.Sprintf("Error: Stream interrupted: %v", readErr)}
				return nil, textBuilder.String(), true
			}
			break
		}
	}

	if len(toolCalls) == 0 && currentToolCall != nil {
		toolCalls = append(toolCalls, *currentToolCall)
	}

	return toolCalls, textBuilder.String(), false
}
