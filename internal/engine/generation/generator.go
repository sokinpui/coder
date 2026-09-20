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

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

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
	Config   config.Generation
	BaseURL  string
	Protocol string
	APIKey   string
}

func New(cfg *config.Config) (*Generator, error) {
	protocol := cfg.Server.Protocol
	if protocol == "" {
		protocol = "responses"
	}
	return &Generator{
		Config:   cfg.Generation,
		BaseURL:  cfg.Server.URL,
		Protocol: protocol,
		APIKey:   cfg.Server.APIKey,
	}, nil
}

func (g *Generator) getChatURL() string {
	return strings.TrimSuffix(g.BaseURL, "/") + "/chat/completions"
}

func (g *Generator) getResponsesURL() string {
	return strings.TrimSuffix(g.BaseURL, "/") + "/responses"
}

func (g *Generator) GenerateTask(ctx context.Context, systemInstruction string, messages []types.ChatMessage, tools []types.ToolDeclaration, streamChan chan<- types.StreamChunk, generationConfig *config.Generation) {
	defer close(streamChan)

	genConfig := g.Config
	if generationConfig != nil {
		genConfig = *generationConfig
	}
	if strings.EqualFold(g.Protocol, "chat") {
		g.generateChatTask(ctx, systemInstruction, messages, tools, streamChan, &genConfig)
		return
	}
	g.generateResponsesTask(ctx, systemInstruction, messages, tools, streamChan, &genConfig)
}

func (g *Generator) generateChatTask(ctx context.Context, systemInstruction string, messages []types.ChatMessage, tools []types.ToolDeclaration, streamChan chan<- types.StreamChunk, genConfig *config.Generation) {
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
		"model":            genConfig.ModelCode,
		"stream":           true,
		"messages":         apiMessages,
		"reasoning_effort": genConfig.ReasoningEffort,
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
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Failed to create request: %v", err)}
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
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Failed to connect to server: %v", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Server returned %d: %s", resp.StatusCode, string(errMsg))}
		return
	}

	type partialToolCall struct {
		id        string
		name      string
		arguments strings.Builder
	}
	var toolCalls []*partialToolCall

	scanner := bufio.NewScanner(resp.Body)
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
			}
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Stream interrupted: %v", err)}
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

func (g *Generator) generateResponsesTask(ctx context.Context, systemInstruction string, messages []types.ChatMessage, tools []types.ToolDeclaration, streamChan chan<- types.StreamChunk, genConfig *config.Generation) {
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
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Failed to create request: %v", err)}
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
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Failed to connect to server: %v", err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Server returned status %d: %s", resp.StatusCode, string(errBytes))}
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
				return
			case "response.incomplete":
				return
			case "error", "response.failed":
				errMsg := data
				if ev.Error != nil && ev.Error.Message != "" {
					errMsg = ev.Error.Message
				}
				streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: %s", errMsg)}
				return
			}
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Stream interrupted: %v", err)}
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

func (g *Generator) GenerateTitle(ctx context.Context, prompt string) (string, error) {
	if strings.EqualFold(g.Protocol, "chat") {
		return g.generateChatTitle(ctx, prompt)
	}
	return g.generateResponsesTitle(ctx, prompt)
}

func (g *Generator) generateChatTitle(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model":  g.Config.TitleModelCode,
		"stream": false,
		"messages": []openAIMessage{
			{Role: "user", Content: prompt},
		},
		"temperature": 1.0,
		"max_tokens":  256,
	}

	jsonBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.getChatURL(), bytes.NewBuffer(jsonBody))
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

	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", err
	}
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("empty choices in title generation")
	}

	str, ok := openAIResp.Choices[0].Message.Content.(string)
	if !ok {
		return "", fmt.Errorf("unexpected content format in title generation")
	}
	return strings.TrimSpace(str), nil
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
