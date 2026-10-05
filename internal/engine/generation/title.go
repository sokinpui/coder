package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (g *Generator) GenerateTitle(ctx context.Context, prompt string) (string, error) {
	if strings.EqualFold(g.Protocol, "chat") {
		return g.generateChatTitle(ctx, prompt)
	}
	return g.generateResponsesTitle(ctx, prompt)
}

func (g *Generator) generateChatTitle(ctx context.Context, prompt string) (string, error) {
	titleModel := g.TitleModel
	if titleModel == "" {
		titleModel = g.Config.ModelCode
	}
	body := map[string]any{
		"model":  titleModel,
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
	titleModel := g.TitleModel
	if titleModel == "" {
		titleModel = g.Config.ModelCode
	}
	body := map[string]any{
		"model":             titleModel,
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
