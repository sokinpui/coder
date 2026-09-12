package generation

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
	"github.com/sokinpui/coder/internal/tools"
	"github.com/sokinpui/coder/internal/types"
)

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type responseStreamEvent struct {
	Type      string `json:"type"`
	Delta     string `json:"delta,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Item      *struct {
		Type      string `json:"type"`
		ID        string `json:"id"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"item,omitempty"`
	Response *struct {
		Output []struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			CallID    string `json:"call_id"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"output"`
	} `json:"response,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type responseNonStreamResponse struct {
	Output []struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type Generator struct {
	Config  config.Generation
	BaseURL string
	APIKey  string
}

func New(cfg *config.Config) (*Generator, error) {
	return &Generator{
		Config:  cfg.Generation,
		BaseURL: cfg.Server.URL,
		APIKey:  cfg.Server.APIKey,
	}, nil
}

func (g *Generator) getResponsesURL() string {
	return strings.TrimSuffix(g.BaseURL, "/") + "/responses"
}

func (g *Generator) GenerateTask(ctx context.Context, messages []types.Message, streamChan chan<- types.StreamChunk, generationConfig *config.Generation) {
	defer close(streamChan)

	genConfig := g.Config
	if generationConfig != nil {
		genConfig = *generationConfig
	}
	g.generateResponsesTask(ctx, messages, streamChan, &genConfig)
}

func (g *Generator) generateResponsesTask(ctx context.Context, messages []types.Message, streamChan chan<- types.StreamChunk, genConfig *config.Generation) {
	currentMessages := append([]types.Message(nil), messages...)
	maxIterations := genConfig.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 20
	}

	for range maxIterations {
		if ctx.Err() != nil {
			return
		}

		toolCalls, turnText, hasError := g.executeTurn(ctx, currentMessages, streamChan, genConfig, genConfig.EnableTools)
		if turnText != "" {
			currentMessages = append(currentMessages, types.Message{Type: types.AIMessage, Content: turnText})
		}
		if hasError || ctx.Err() != nil || len(toolCalls) == 0 {
			return
		}

		for _, tc := range toolCalls {
			if ctx.Err() != nil {
				return
			}

			streamChan <- types.StreamChunk{ToolCall: &tc}
			output, err := tools.DefaultRegistry.Execute(ctx, tc.Name, tc.Arguments)
			if err != nil {
				output = fmt.Sprintf("Error: %v", err)
			}

			resInfo := types.ToolResultInfo{
				CallID: tc.CallID,
				Name:   tc.Name,
				Output: output,
			}
			streamChan <- types.StreamChunk{ToolResult: &resInfo}

			currentMessages = append(currentMessages,
				types.Message{
					Type:     types.ToolCallMessage,
					CallID:   tc.CallID,
					ToolName: tc.Name,
					Content:  tc.Arguments,
				},
				types.Message{
					Type:     types.ToolCallResultMessage,
					CallID:   tc.CallID,
					ToolName: tc.Name,
					Content:  output,
				},
			)
		}
	}

	if ctx.Err() != nil {
		return
	}

	_, _, _ = g.executeTurn(ctx, currentMessages, streamChan, genConfig, false)
}

func (g *Generator) executeTurn(ctx context.Context, messages []types.Message, streamChan chan<- types.StreamChunk, genConfig *config.Generation, enableTools bool) ([]types.ToolCallInfo, string, bool) {
	var instructionsBuilder strings.Builder
	var inputItems []any

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}

		switch msg.Type {
		case types.InstructionMessage, types.DirectoryMessage, types.SourceCodeMessage:
			if instructionsBuilder.Len() > 0 {
				instructionsBuilder.WriteString("\n\n")
			}
			instructionsBuilder.WriteString(msg.Content)

		case types.UserMessage, types.ShellCmdMessage, types.ShellCmdResultMessage,
			types.ContextCmdMessage, types.ContextCmdResultMessage,
			types.FileApplyCmdMessage, types.FileApplyCmdResultMessage, types.FileApplyCmdErrorMessage,
			types.FileApplyUndoCmdMessage, types.FileApplyUndoCmdResultMessage, types.FileApplyUndoCmdErrorMessage:
			inputItems = append(inputItems, map[string]any{
				"type":    "message",
				"role":    "user",
				"content": msg.Content,
			})

		case types.AIMessage:
			if msg.Content != "" {
				inputItems = append(inputItems, map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": msg.Content,
				})
			}

		case types.ToolCallMessage:
			callID := msg.CallID
			if callID == "" {
				callID = "call_fallback"
			}
			inputItems = append(inputItems, map[string]any{
				"type":      "function_call",
				"call_id":   callID,
				"name":      msg.ToolName,
				"arguments": msg.Content,
			})

		case types.ToolCallResultMessage:
			inputItems = append(inputItems, map[string]any{
				"type":    "function_call_output",
				"call_id": msg.CallID,
				"output":  msg.Content,
			})

		case types.ImageMessage:
			if msg.Data == nil {
				continue
			}
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
		}
	}

	if genConfig.EnableTools && !enableTools {
		inputItems = append(inputItems, map[string]any{
			"type":    "message",
			"role":    "user",
			"content": "Tool execution iteration limit reached. Do not call any tools. Provide your final response to the user based on the tool results so far.",
		})
	}

	body := map[string]any{
		"model":        genConfig.ModelCode,
		"stream":       true,
		"store":        false,
		"instructions": instructionsBuilder.String(),
		"input":        inputItems,
	}

	if genConfig.EnableTools {
		toolDecls := tools.DefaultRegistry.Declarations()
		if len(toolDecls) > 0 {
			body["tools"] = toolDecls
			if enableTools {
				body["tool_choice"] = "auto"
			} else {
				body["tool_choice"] = "none"
			}
		}
	}

	if genConfig.ReasoningEffort != "" {
		body["reasoning"] = map[string]any{
			"effort": genConfig.ReasoningEffort,
		}
	}

	jsonBody, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.getResponsesURL(), bytes.NewBuffer(jsonBody))
	if err != nil {
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Failed to create request: %v", err)}
		return nil, "", true
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if g.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, "", false
		}
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Failed to connect to server: %v", err)}
		return nil, "", true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Server returned status %d: %s", resp.StatusCode, string(errBytes))}
		return nil, "", true
	}

	var toolCalls []types.ToolCallInfo
	var currentToolCall *types.ToolCallInfo
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
							case streamChan <- types.StreamChunk{Content: ev.Delta}:
							}
						}
					case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
						if ev.Delta != "" {
							select {
							case <-ctx.Done():
								return nil, textBuilder.String(), false
							case streamChan <- types.StreamChunk{ReasoningContent: ev.Delta}:
							}
						}
					case "response.output_item.added":
						if ev.Item != nil && ev.Item.Type == "function_call" {
							currentToolCall = &types.ToolCallInfo{
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
							toolCalls = append(toolCalls, types.ToolCallInfo{
								CallID:    cid,
								Name:      ev.Item.Name,
								Arguments: args,
							})
							currentToolCall = nil
						}
					case "response.completed":
						if len(toolCalls) == 0 && ev.Response != nil {
							for _, out := range ev.Response.Output {
								if out.Type == "function_call" {
									cid := out.CallID
									if cid == "" {
										cid = out.ID
									}
									toolCalls = append(toolCalls, types.ToolCallInfo{
										CallID:    cid,
										Name:      out.Name,
										Arguments: out.Arguments,
									})
								}
							}
						}
						return toolCalls, textBuilder.String(), false
					case "response.incomplete":
						return toolCalls, textBuilder.String(), false
					case "error", "response.failed":
						errMsg := trimmed
						if ev.Error != nil && ev.Error.Message != "" {
							errMsg = ev.Error.Message
						}
						streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: %s", errMsg)}
						return nil, textBuilder.String(), true
					}
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF && ctx.Err() == nil {
				streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Stream interrupted: %v", readErr)}
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

func (g *Generator) GenerateTitle(ctx context.Context, prompt string) (string, error) {
	return g.generateResponsesTitle(ctx, prompt)
}

func (g *Generator) generateResponsesTitle(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model":             g.Config.TitleModelCode,
		"stream":            false,
		"store":             false,
		"instructions":      "You are an expert in summarizing conversations. Create a short, concise title (5-10 words) for the prompt. Do not add quotes or prefixes like 'Title:'.",
		"input":             prompt,
		"max_output_tokens": 256,
		"temperature":       1.0,
	}

	jsonBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.getResponsesURL(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if g.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server error %d: %s", resp.StatusCode, string(errMsg))
	}

	var respObj responseNonStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&respObj); err != nil {
		return "", err
	}

	if respObj.Error != nil {
		return "", fmt.Errorf("API error: %s", respObj.Error.Message)
	}

	for _, out := range respObj.Output {
		for _, part := range out.Content {
			if part.Type == "output_text" && strings.TrimSpace(part.Text) != "" {
				return strings.Trim(strings.TrimSpace(part.Text), "\""), nil
			}
		}
	}
	return "", fmt.Errorf("empty text output in responses title response")
}
