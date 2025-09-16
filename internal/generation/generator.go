package generation

import (
	"coder/internal/config"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	pb "coder/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Generator handles communication with the code generation gRPC service.
type Generator struct {
	client pb.GenerateClient
	Config config.Generation
}

// New creates a new Generator.
func New(cfg *config.Config) (*Generator, error) {
	conn, err := grpc.Dial(cfg.GRPC.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect: %w", err)
	}

	client := pb.NewGenerateClient(conn)
	return &Generator{client: client, Config: cfg.Generation}, nil
}

// GenerateTask sends a prompt to the generation service and streams the response.
func (g *Generator) GenerateTask(ctx context.Context, prompt string, streamChan chan<- string) {
	defer close(streamChan)

	req := &pb.Request{
		Prompt:    prompt,
		ModelCode: g.Config.ModelCode,
		Stream:    true,
		Config: &pb.GenerationConfig{
			Temperature:  &g.Config.Temperature,
			TopP:         &g.Config.TopP,
			TopK:         &g.Config.TopK,
			OutputLength: &g.Config.OutputLength,
		},
	}

	stream, err := g.client.GenerateTask(ctx, req)
	if err != nil {
		log.Printf("GenerateTask failed: %v", err)
		streamChan <- fmt.Sprintf("Error: Could not connect to generation service: %v", err)
		return
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// If context is cancelled, grpc will return a status error with code Canceled.
			if ctx.Err() != nil {
				log.Printf("Generation cancelled: %v", ctx.Err())
				break
			}
			log.Printf("Stream recv failed: %v", err)
			streamChan <- fmt.Sprintf("Error: Stream failed: %v", err)
			break
		}
		chunk := resp.GetOutputString()
		log.Printf("Received raw chunk from server: %q", chunk)
		streamChan <- chunk
	}
}

// GenerateTitle sends a prompt to the generation service and gets a single response for a title.
func (g *Generator) GenerateTitle(ctx context.Context, prompt string) (string, error) {
	// A smaller output length for titles.
	outputLength := int32(256)
	temp := float32(0.2) // A bit of creativity for titles

	req := &pb.Request{
		Prompt:    prompt,
		ModelCode: "gemini-2.0-flash-lite",
		Stream:    false, // We want a single response
		Config: &pb.GenerationConfig{
			Temperature:  &temp,
			TopP:         &g.Config.TopP,
			TopK:         &g.Config.TopK,
			OutputLength: &outputLength,
		},
	}

	stream, err := g.client.GenerateTask(ctx, req)
	if err != nil {
		return "", fmt.Errorf("GenerateTitle failed: %w", err)
	}

	// For non-streaming, we expect one response, then EOF.
	var fullResponse strings.Builder
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("stream recv failed during title generation: %w", err)
		}
		fullResponse.WriteString(resp.GetOutputString())
	}

	return strings.TrimSpace(fullResponse.String()), nil
}
