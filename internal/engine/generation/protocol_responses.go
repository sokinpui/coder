package generation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

type responseStreamEvent struct {
	Type      string          `json:"type"`
	Delta     string          `json:"delta,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Item      *struct {
		Type      string          `json:"type"`
		ID        string          `json:"id"`
		CallID    string          `json:"call_id"`
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"item,omitempty"`
	Response *struct {
		Output []struct {
			Type      string          `json:"type"`
			ID        string          `json:"id"`
			CallID    string          `json:"call_id"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"output"`
	} `json:"response,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (g *Generator) generateResponsesTask(ctx context.Context, systemInstruction string, messages []types.ChatMessage, tools []types.ToolDeclaration, streamChan chan<- types.StreamChunk, genConfig *config.ModelConfig) {
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
			if msg.Content == "" {
				continue
			}
			inputItems = append(inputItems, map[string]any{
				"type":    "message",
				"role":    "user",
				"content": msg.Content,
			})
		case types.RoleAssistant:
			if msg.Content != "" {
				inputItems = append(inputItems, map[string]any{
					"type":    "message",
					"role":    "assistant",
					"content": msg.Content,
				})
			}
			for _, tc := range msg.ToolCalls {
				inputItems = append(inputItems, map[string]any{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      tc.Name,
					"arguments": tc.Arguments,
				})
			}
		case types.RoleTool:
			inputItems = append(inputItems, map[string]any{
				"type":    "function_call_output",
				"call_id": msg.ToolCallID,
				"output":  msg.Content,
			})
		}
	}

	body := map[string]any{
		"model":        genConfig.ModelCode,
		"stream":       true,
		"store":        false,
		"instructions": systemInstruction,
		"input":        inputItems,
	}

	if len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}

	if genConfig.ReasoningEffort != "" {
		body["reasoning"] = map[string]any{
			"effort": genConfig.ReasoningEffort,
		}
	}

	jsonBody, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.getResponsesURL(), bytes.NewBuffer(jsonBody))
	if err != nil {
		streamChan <- types.StreamChunk{Error: fmt.Errorf("failed to create request: %w", err)}
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if g.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return
		}
		streamChan <- types.StreamChunk{Error: fmt.Errorf("failed to connect to server: %w", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		streamChan <- types.StreamChunk{Error: fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(errBytes))}
		return
	}

	type partialToolCall struct {
		id        string
		name      string
		arguments strings.Builder
	}
	var toolCalls []*partialToolCall
	var currentToolCall *partialToolCall
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, initialStreamBufferSize), maxStreamBufferSize)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}

		line := strings.TrimSpace(scanner.Text())
		after, isData := strings.CutPrefix(line, "data:")
		if !isData {
			continue
		}

		data := strings.TrimSpace(after)
		if data == "[DONE]" {
			break
		}

		var ev responseStreamEvent
		if err := json.Unmarshal([]byte(data), &ev); err == nil {
			switch ev.Type {
			case "response.output_text.delta":
				if ev.Delta != "" {
					select {
					case <-ctx.Done():
						return
					case streamChan <- types.StreamChunk{Content: ev.Delta}:
					}
				}
			case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
				if ev.Delta != "" {
					select {
					case <-ctx.Done():
						return
					case streamChan <- types.StreamChunk{ReasoningContent: ev.Delta}:
					}
				}
			case "response.output_item.added":
				if ev.Item != nil && ev.Item.Type == "function_call" {
					cid := ev.Item.CallID
					if cid == "" {
						cid = ev.Item.ID
					}
					currentToolCall = &partialToolCall{
						id:   cid,
						name: ev.Item.Name,
					}
					currentToolCall.arguments.WriteString(parseRawArgs(ev.Item.Arguments))
				}
			case "response.function_call_arguments.delta":
				if currentToolCall != nil {
					currentToolCall.arguments.WriteString(ev.Delta)
				}
			case "response.function_call_arguments.done":
				if currentToolCall != nil {
					args := parseRawArgs(ev.Arguments)
					if args != "" {
						currentToolCall.arguments.Reset()
						currentToolCall.arguments.WriteString(args)
					}
				}
			case "response.output_item.done":
				if ev.Item != nil && ev.Item.Type == "function_call" {
					cid := ev.Item.CallID
					if cid == "" {
						cid = ev.Item.ID
					}
					name := ev.Item.Name
					args := parseRawArgs(ev.Item.Arguments)
					if currentToolCall != nil {
						if name == "" {
							name = currentToolCall.name
						}
						if args == "" {
							args = currentToolCall.arguments.String()
						}
						if cid == "" {
							cid = currentToolCall.id
						}
					}
					tc := &partialToolCall{
						id:   cid,
						name: name,
					}
					tc.arguments.WriteString(args)
					toolCalls = append(toolCalls, tc)
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
							tc := &partialToolCall{
								id:   cid,
								name: out.Name,
							}
							tc.arguments.WriteString(parseRawArgs(out.Arguments))
							toolCalls = append(toolCalls, tc)
						}
					}
				}
				break
			case "response.incomplete":
				break
			case "error", "response.failed":
				errMsg := data
				if ev.Error != nil && ev.Error.Message != "" {
					errMsg = ev.Error.Message
				}
				streamChan <- types.StreamChunk{Error: errors.New(errMsg)}
				return
			}
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		streamChan <- types.StreamChunk{Error: fmt.Errorf("stream interrupted: %w", err)}
	}

	if len(toolCalls) == 0 && currentToolCall != nil {
		toolCalls = append(toolCalls, currentToolCall)
	}

	for _, tc := range toolCalls {
		if tc.name == "" && tc.id == "" && tc.arguments.Len() == 0 {
			continue
		}
		id := tc.id
		if id == "" {
			id = fmt.Sprintf("call_%d", time.Now().UnixNano())
		}
		select {
		case <-ctx.Done():
			return
		case streamChan <- types.StreamChunk{
			ToolCall: &types.ToolCall{
				ID:        id,
				Name:      tc.name,
				Arguments: tc.arguments.String(),
			},
		}:
		}
	}
}

func parseRawArgs(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str
	}
	return string(raw)
}
