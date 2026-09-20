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

type openAIMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
	} `json:"choices"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

type responseStreamEvent struct {
	Type      string `json:"type"`
	Delta     string `json:"delta,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Error     *struct {
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

func (g *Generator) GenerateTask(ctx context.Context, messages []types.Message, streamChan chan<- types.StreamChunk, generationConfig *config.Generation) {
	defer close(streamChan)

	genConfig := g.Config
	if generationConfig != nil {
		genConfig = *generationConfig
	}
	if strings.EqualFold(g.Protocol, "chat") {
		g.generateChatTask(ctx, messages, streamChan, &genConfig)
		return
	}
	g.generateResponsesTask(ctx, messages, streamChan, &genConfig)
}

func (g *Generator) generateChatTask(ctx context.Context, messages []types.Message, streamChan chan<- types.StreamChunk, genConfig *config.Generation) {
	var apiMessages []openAIMessage
	var sourceCodeParts []openAIContentPart

	flushSourceParts := func() {
		if len(sourceCodeParts) == 0 {
			return
		}
		apiMessages = append(apiMessages, openAIMessage{
			Role:    "user",
			Content: sourceCodeParts,
		})
		sourceCodeParts = nil
	}

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}

		if msg.Type == types.SourceCodeMessage {
			if strings.TrimSpace(msg.Content) == "" {
				continue
			}
			sourceCodeParts = append(sourceCodeParts, openAIContentPart{
				Type: "text",
				Text: msg.Content,
			})
			continue
		}

		flushSourceParts()

		role := ""
		var content any

		switch msg.Type {
		case types.InstructionMessage, types.DirectoryMessage:
			role = "system"
			content = msg.Content
		case types.UserMessage, types.ShellCmdMessage, types.ShellCmdResultMessage,
			types.ContextCmdMessage, types.ContextCmdResultMessage,
			types.FileApplyCmdMessage, types.FileApplyCmdResultMessage, types.FileApplyCmdErrorMessage,
			types.FileApplyUndoCmdMessage, types.FileApplyUndoCmdResultMessage, types.FileApplyUndoCmdErrorMessage:
			role = "user"
			content = msg.Content
		case types.AIMessage:
			role = "assistant"
			content = msg.Content
		case types.ImageMessage:
			if msg.Data == nil {
				continue
			}
			role = "user"
			b64 := base64.StdEncoding.EncodeToString(msg.Data)
			mimeType := "image/png"
			if len(msg.Data) > 4 && bytes.Equal(msg.Data[:4], []byte{0xFF, 0xD8, 0xFF, 0xE0}) {
				mimeType = "image/jpeg"
			}
			content = []openAIContentPart{
				{
					Type: "image_url",
					ImageURL: &openAIImageURL{
						URL: fmt.Sprintf("data:%s;base64,%s", mimeType, b64),
					},
				},
			}
		default:
			continue
		}

		if role == "" || (content == "" && msg.Type != types.ImageMessage) {
			continue
		}

		if len(apiMessages) > 0 && apiMessages[len(apiMessages)-1].Role == role {
			prevContent, isPrevStr := apiMessages[len(apiMessages)-1].Content.(string)
			currContent, isCurrStr := content.(string)
			if isPrevStr && isCurrStr {
				apiMessages[len(apiMessages)-1].Content = prevContent + "\n\n" + currContent
				continue
			}
		}

		apiMessages = append(apiMessages, openAIMessage{
			Role:    role,
			Content: content,
		})
	}

	flushSourceParts()

	body := map[string]any{
		"model":            genConfig.ModelCode,
		"stream":           true,
		"messages":         apiMessages,
		"reasoning_effort": genConfig.ReasoningEffort,
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

	reader := bufio.NewReader(resp.Body)
	for {
		if ctx.Err() != nil {
			return
		}

		line, readErr := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if after, ok := strings.CutPrefix(line, "data:"); ok {
				data := strings.TrimPrefix(after, " ")
				if strings.TrimSpace(data) == "[DONE]" {
					break
				}
				var streamResp openAIStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err == nil && len(streamResp.Choices) > 0 {
					delta := streamResp.Choices[0].Delta
					if delta.Content != "" || delta.ReasoningContent != "" {
						select {
						case <-ctx.Done():
							return
						case streamChan <- types.StreamChunk{
							Content:          delta.Content,
							ReasoningContent: delta.ReasoningContent,
						}:
						}
					}
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF && ctx.Err() == nil {
				streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Stream interrupted: %v", readErr)}
			}
			break
		}
	}
}

func (g *Generator) generateResponsesTask(ctx context.Context, messages []types.Message, streamChan chan<- types.StreamChunk, genConfig *config.Generation) {
	var instructionsBuilder strings.Builder
	var inputItems []any

	for _, msg := range messages {
		if !msg.CanSendToAI() {
			continue
		}

		if msg.Type == types.SourceCodeMessage {
			if strings.TrimSpace(msg.Content) == "" {
				continue
			}
			inputItems = append(inputItems, map[string]any{
				"type":    "message",
				"role":    "user",
				"content": msg.Content,
			})
			continue
		}

		switch msg.Type {
		case types.InstructionMessage, types.DirectoryMessage:
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

	body := map[string]any{
		"model":        genConfig.ModelCode,
		"stream":       true,
		"store":        false,
		"instructions": instructionsBuilder.String(),
		"input":        inputItems,
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

	reader := bufio.NewReader(resp.Body)
	for {
		if ctx.Err() != nil {
			return
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
					case "response.completed", "response.incomplete":
						return
					case "error", "response.failed":
						errMsg := trimmed
						if ev.Error != nil && ev.Error.Message != "" {
							errMsg = ev.Error.Message
						}
						streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: %s", errMsg)}
						return
					}
				}
			}
		}

		if readErr != nil {
			if readErr != io.EOF && ctx.Err() == nil {
				streamChan <- types.StreamChunk{Content: fmt.Sprintf("Error: Stream interrupted: %v", readErr)}
			}
			break
		}
	}
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
