package generation

import (
	"coder/internal/config"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/sokinpui/synapse.go/client"
)

// Generator handles communication with the code generation gRPC service.
type Generator struct {
	client client.Client
	Config config.Generation
}

// New creates a new Generator.
func New(cfg *config.Config) (*Generator, error) {
	c, err := client.New(cfg.GRPC.Addr)
	if err != nil {
		return nil, fmt.Errorf("could not create synapse client: %w", err)
	}
	return &Generator{client: c, Config: cfg.Generation}, nil
}

// GenerateTask sends a prompt to the generation service and streams the response.
func (g *Generator) GenerateTask(ctx context.Context, prompt string, images [][]byte, streamChan chan<- string, generationConfig *config.Generation) {
	defer close(streamChan)

	// Use generator's default config if none is provided.
	genConfig := g.Config
	if generationConfig != nil {
		genConfig = *generationConfig
	}

	// Convert to the client's generation config type.
	clientConfig := &client.GenerationConfig{
		Temperature:  &genConfig.Temperature,
		TopP:         &genConfig.TopP,
		TopK:         &genConfig.TopK,
		OutputLength: &genConfig.OutputLength,
	}

	req := &client.GenerateRequest{
		Prompt:    prompt,
		Images:    images,
		ModelCode: genConfig.ModelCode,
		Stream:    true,
		Config:    clientConfig,
	}

	log.Printf("Generating with model: %s", genConfig.ModelCode)

	resultChan, err := g.client.GenerateTask(ctx, req)
	if err != nil {
		log.Printf("GenerateTask failed: %v", err)
		streamChan <- fmt.Sprintf("Error: Could not connect to generation service: %v", err)
		return
	}

	for result := range resultChan {
		if result.Err != nil {
			// If context is cancelled, the client will return a context error.
			if ctx.Err() != nil {
				log.Printf("Generation cancelled: %v", ctx.Err())
				break
			}
			log.Printf("Stream recv failed: %v", result.Err)
			streamChan <- fmt.Sprintf("Error: Stream failed: %v", result.Err)
			break
		}
		if result.IsKeepAlive {
			log.Printf("Received keep-alive from server")
			continue
		}
		log.Printf("Received raw chunk from server: %q", result.Text)
		streamChan <- result.Text
	}
}

// GenerateTitle sends a prompt to the generation service and gets a single response for a title.
func (g *Generator) GenerateTitle(ctx context.Context, prompt string) (string, error) {
	// A smaller output length for titles.
	outputLength := int32(256)
	temp := float32(1.0)

	req := &client.GenerateRequest{
		Prompt:    prompt,
		ModelCode: g.Config.TitleModelCode,
		Stream:    false,
		Config: &client.GenerationConfig{
			Temperature:  &temp,
			TopP:         &g.Config.TopP,
			TopK:         &g.Config.TopK,
			OutputLength: &outputLength,
		},
	}

	log.Printf("Generating title with model: %s", req.ModelCode)

	resultChan, err := g.client.GenerateTask(ctx, req)
	if err != nil {
		return "", fmt.Errorf("GenerateTitle failed: %w", err)
	}

	var fullResponse strings.Builder
	for result := range resultChan {
		if result.Err != nil {
			return "", fmt.Errorf("stream recv failed during title generation: %w", result.Err)
		}
		fullResponse.WriteString(result.Text)
	}

	return strings.TrimSpace(fullResponse.String()), nil
}
