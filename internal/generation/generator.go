package generation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sokinpui/coder/internal/config"
)

type Generator struct {
	Config     config.Generation
	ServerAddr string
}

func New(cfg *config.Config) (*Generator, error) {
	return &Generator{
		Config:     cfg.Generation,
		ServerAddr: strings.TrimSuffix(cfg.Server.Addr, "/"),
	}, nil
}

func (g *Generator) GenerateTask(ctx context.Context, prompt string, images [][]byte, streamChan chan<- string, generationConfig *config.Generation) {
	defer close(streamChan)

	genConfig := g.Config
	if generationConfig != nil {
		genConfig = *generationConfig
	}

	body := map[string]interface{}{
		"prompt":     prompt,
		"model_code": genConfig.ModelCode,
		"stream":     true,
		"images":     images,
		"config": map[string]interface{}{
			"temperature":   genConfig.Temperature,
			"top_p":         genConfig.TopP,
			"top_k":         genConfig.TopK,
			"output_length": genConfig.OutputLength,
		},
	}

	jsonBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.ServerAddr+"/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		streamChan <- fmt.Sprintf("Error: Failed to create request: %v", err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var result struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			continue
		}
		if result.Text != "" {
			streamChan <- result.Text
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		streamChan <- fmt.Sprintf("Error: Stream interrupted: %v", err)
	}
}

func (g *Generator) GenerateTitle(ctx context.Context, prompt string) (string, error) {
	body := map[string]interface{}{
		"prompt":     prompt,
		"model_code": g.Config.TitleModelCode,
		"stream":     false,
		"config": map[string]interface{}{
			"temperature":   1.0,
			"output_length": 256,
		},
	}

	jsonBody, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.ServerAddr+"/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Text), nil
}
