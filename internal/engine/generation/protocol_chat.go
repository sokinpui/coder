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
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

type openAIToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAIToolCall struct {
	Index    int                    `json:"index,omitempty"`
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Function openAIToolCallFunction `json:"function"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content          string           `json:"content"`
			ReasoningContent string           `json:"reasoning_content"`
			Reasoning        string           `json:"reasoning"`
			ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		Message struct {
			Content          string           `json:"content"`
			ReasoningContent string           `json:"reasoning_content"`
			ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (g *Generator) generateChatTask(ctx context.Context, systemInstruction string, messages []types.ChatMessage, tools []types.ToolDeclaration, streamChan chan<- types.StreamChunk, genConfig *config.ModelConfig) {
	var apiMessages []openAIMessage
	if systemInstruction != "" {
		apiMessages = append(apiMessages, openAIMessage{
			Role:    "system",
			Content: systemInstruction,
		})
	}

	for _, msg := range messages {
		switch msg.Role {
		case types.RoleUser:
			if len(msg.Data) > 0 {
				b64 := base64.StdEncoding.EncodeToString(msg.Data)
				mimeType := "image/png"
				if len(msg.Data) > 4 && bytes.Equal(msg.Data[:4], []byte{0xFF, 0xD8, 0xFF, 0xE0}) {
					mimeType = "image/jpeg"
				}
				apiMessages = append(apiMessages, openAIMessage{
					Role: "user",
					Content: []openAIContentPart{
						{
							Type: "image_url",
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
			if len(apiMessages) > 0 && apiMessages[len(apiMessages)-1].Role == "user" {
				if prevStr, ok := apiMessages[len(apiMessages)-1].Content.(string); ok {
					apiMessages[len(apiMessages)-1].Content = prevStr + "\n\n" + msg.Content
					continue
				}
			}
			apiMessages = append(apiMessages, openAIMessage{
				Role:    "user",
				Content: msg.Content,
			})
		case types.RoleAssistant:
			if msg.Content == "" && len(msg.ToolCalls) == 0 {
				continue
			}
			assistantMsg := openAIMessage{
				Role: "assistant",
			}
			if msg.Content != "" {
				assistantMsg.Content = msg.Content
			}
			for _, tc := range msg.ToolCalls {
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, openAIToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openAIToolCallFunction{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
			apiMessages = append(apiMessages, assistantMsg)
		case types.RoleTool:
			callID := msg.ToolCallID
			if callID == "" {
				callID = "call_fallback"
			}
			apiMessages = append(apiMessages, openAIMessage{
				Role:       "tool",
				Content:    msg.Content,
				ToolCallID: callID,
			})
		case types.RoleSystem:
			if msg.Content == "" {
				continue
			}
			apiMessages = append(apiMessages, openAIMessage{
				Role:    "system",
				Content: msg.Content,
			})
		}
	}

	body := map[string]any{
		"model":    genConfig.ModelCode,
		"stream":   true,
		"messages": apiMessages,
	}

	if genConfig.ReasoningEffort != "" {
		body["reasoning_effort"] = genConfig.ReasoningEffort
	}

	if len(tools) > 0 {
		chatTools := make([]map[string]any, 0, len(tools))
		for _, t := range tools {
			toolDef := map[string]any{
				"name":        t.Name,
				"description": t.Description,
			}
			if t.Parameters != nil {
				toolDef["parameters"] = t.Parameters
			}
			chatTools = append(chatTools, map[string]any{
				"type":     "function",
				"function": toolDef,
			})
		}
		body["tools"] = chatTools
		body["tool_choice"] = "auto"
	}

	jsonBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.getChatURL(), bytes.NewBuffer(jsonBody))
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
		errMsg, _ := io.ReadAll(resp.Body)
		streamChan <- types.StreamChunk{Error: fmt.Errorf("server returned %d: %s", resp.StatusCode, string(errMsg))}
		return
	}

	type partialToolCall struct {
		id        string
		name      string
		arguments strings.Builder
	}
	var toolCalls []*partialToolCall

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

		var streamResp openAIStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err == nil && len(streamResp.Choices) > 0 {
			choice := streamResp.Choices[0]
			delta := choice.Delta
			reasoning := delta.ReasoningContent
			if reasoning == "" {
				reasoning = delta.Reasoning
			}
			if delta.Content != "" || reasoning != "" {
				select {
				case <-ctx.Done():
					return
				case streamChan <- types.StreamChunk{
					Content:          delta.Content,
					ReasoningContent: reasoning,
				}:
				}
			}
			rawTCs := delta.ToolCalls
			if len(rawTCs) == 0 && len(choice.Message.ToolCalls) > 0 {
				rawTCs = choice.Message.ToolCalls
			}
			for i, tcDelta := range rawTCs {
				idx := tcDelta.Index
				if idx == 0 && i > 0 {
					idx = i
				}
				if idx < 0 {
					continue
				}
				for len(toolCalls) <= idx {
					toolCalls = append(toolCalls, &partialToolCall{})
				}
				if tcDelta.ID != "" {
					toolCalls[idx].id = tcDelta.ID
				}
				if tcDelta.Function.Name != "" {
					toolCalls[idx].name = tcDelta.Function.Name
				}
				if tcDelta.Function.Arguments != "" {
					toolCalls[idx].arguments.WriteString(tcDelta.Function.Arguments)
				}

				select {
				case <-ctx.Done():
					return
				case streamChan <- types.StreamChunk{
					ToolCallDelta: &types.ToolCallDelta{
						Index:          idx,
						ID:             tcDelta.ID,
						Name:           tcDelta.Function.Name,
						ArgumentsDelta: tcDelta.Function.Arguments,
					},
				}:
				}
			}
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		streamChan <- types.StreamChunk{Error: fmt.Errorf("stream interrupted: %w", err)}
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
