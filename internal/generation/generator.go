package generation

import (
	"coder/internal/config"
	"context"
	"fmt"
	"io"
	"log"

	pb "coder/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Generator handles communication with the code generation gRPC service.
type Generator struct {
	client pb.GenerateClient
	config config.Generation
}

// New creates a new Generator.
func New(cfg *config.Config) (*Generator, error) {
	conn, err := grpc.Dial(cfg.GRPC.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect: %w", err)
	}

	client := pb.NewGenerateClient(conn)
	return &Generator{client: client, config: cfg.Generation}, nil
}

// GenerateTask sends a prompt to the generation service and streams the response.
func (g *Generator) GenerateTask(ctx context.Context, prompt string, streamChan chan<- string) {
	defer close(streamChan)

	req := &pb.Request{
		Prompt:    prompt,
		ModelCode: g.config.ModelCode,
		Stream:    true,
		Config: &pb.GenerationConfig{
			Temperature:  &g.config.Temperature,
			TopP:         &g.config.TopP,
			TopK:         &g.config.TopK,
			OutputLength: &g.config.OutputLength,
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
			} else {
				log.Printf("Stream recv failed: %v", err)
				streamChan <- fmt.Sprintf("Error: Stream failed: %v", err)
			}
			break
		}
		streamChan <- resp.GetOutputString()
	}
}
