package generation

import (
	"context"
	"io"
	"log"

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

	req := &pb.Request{
		Prompt:    prompt,
		ModelCode: modelCode,
		Stream:    true,
	}

	stream, err := g.client.GenerateTask(context.Background(), req)
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
