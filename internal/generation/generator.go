package generation

import (
	"context"
	"io"
	"log"
	"time"

	pb "coder/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverAddr = "localhost:50051"
	modelCode  = "gemini-2.0-flash-lite"
)

type Generator struct {
	client pb.GenerateClient
}

func New() *Generator {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	client := pb.NewGenerateClient(conn)
	return &Generator{client: client}
}

func (g *Generator) GenerateTask(prompt string, streamChan chan<- string) {
	defer close(streamChan)

	temp := float32(1.0)
	topP := float32(0.95)
	topK := int32(40)
	outputLength := int32(65536)

	req := &pb.Request{
		Prompt:    prompt,
		ModelCode: modelCode,
		Stream:    true,
		Config: &pb.GenerationConfig{
			Temperature:  &temp,
			TopP:         &topP,
			TopK:         &topK,
			OutputLength: &outputLength,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := g.client.GenerateTask(ctx, req)
	if err != nil {
		log.Printf("GenerateTask failed: %v", err)
		streamChan <- "Error: Could not connect to generation service."
		return
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Stream recv failed: %v", err)
			streamChan <- "Error: Stream failed."
			break
		}
		streamChan <- resp.GetOutputString()
	}
}
