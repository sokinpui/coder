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

type openAIMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

type openAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *openAIImageURL `json:"image_url,omitempty"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

type Generator struct {
	Config     config.Generation
	BaseURL    string
	APIKey     string
}

func New(cfg *config.Config) (*Generator, error) {
	return &Generator{
		Config:     cfg.Generation,
		BaseURL:    cfg.Server.URL,
		APIKey:     cfg.Server.APIKey,
	}, nil
}

func (g *Generator) getChatURL() string {
	return strings.TrimSuffix(g.BaseURL, "/") + "/chat/completions"
}

func (g *Generator) GenerateTask(ctx context.Context, messages []types.Message, streamChan chan<- string, generationConfig *config.Generation) {
	defer close(streamChan)

	genConfig := g.Config
	if generationConfig != nil {
		genConfig = *generationConfig
	}

	var apiMessages []openAIMessage
	for _, msg := range messages {
		role := ""
		var content any

		switch msg.Type {
		case types.InitMessage:
			role = "system"
			content = msg.Content
		case types.UserMessage:
			role = "user"
			content = msg.Content
		case types.AIMessage:
			role = "assistant"
			content = msg.Content
		case types.ImageMessage:
			role = "user"
			if msg.Data == nil {
				continue
			}
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

		if role == "" || content == "" && msg.Type != types.ImageMessage {
			continue
		}

		// Collapse consecutive messages of the same role if they are simple text
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

	body := map[string]any{
		"model":       genConfig.ModelCode,
		"stream":      true,
		"messages":    apiMessages,
		"temperature": genConfig.Temperature,
		"top_p":       genConfig.TopP,
		"max_tokens":  genConfig.OutputLength,
	}

	jsonBody, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.getChatURL(), bytes.NewBuffer(jsonBody))
	if err != nil {
		streamChan <- fmt.Sprintf("Error: Failed to create request: %v", err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if g.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		streamChan <- fmt.Sprintf("Error: Failed to connect to server: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		streamChan <- fmt.Sprintf("Error: Server returned %d: %s", resp.StatusCode, string(errMsg))
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "[DONE]" {
			break
		}

		var streamResp openAIStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		if text := streamResp.Choices[0].Delta.Content; text != "" {
			streamChan <- text
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		streamChan <- fmt.Sprintf("Error: Stream interrupted: %v", err)
	}
}

func (g *Generator) GenerateTitle(ctx context.Context, prompt string) (string, error) {
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

	return strings.TrimSpace(openAIResp.Choices[0].Message.Content.(string)), nil
}
