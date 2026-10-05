package generation

import (
	"context"
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

const (
	initialStreamBufferSize = 64 * 1024
	maxStreamBufferSize     = 10 * 1024 * 1024
)

type Generator struct {
	Config     config.ModelConfig
	TitleModel string
	BaseURL    string
	Protocol   string
	APIKey     string
}

func New(cfg *config.Config) (*Generator, error) {
	protocol := cfg.Server.Protocol
	if protocol == "" {
		protocol = "responses"
	}
	return &Generator{
		Config:     cfg.Coder.ModelConfig(),
		TitleModel: cfg.Title.ModelCode,
		BaseURL:    cfg.Server.URL,
		Protocol:   protocol,
		APIKey:     cfg.Server.APIKey,
	}, nil
}

func (g *Generator) getChatURL() string {
	return strings.TrimSuffix(g.BaseURL, "/") + "/chat/completions"
}

func (g *Generator) getResponsesURL() string {
	return strings.TrimSuffix(g.BaseURL, "/") + "/responses"
}

func (g *Generator) GenerateTask(ctx context.Context, systemInstruction string, messages []types.ChatMessage, tools []types.ToolDeclaration, streamChan chan<- types.StreamChunk, generationConfig *config.ModelConfig) {
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
