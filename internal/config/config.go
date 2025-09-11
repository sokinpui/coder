package config

var AvailableModels = []string{
	"gemini-2.5-pro",
	"gemini-2.5-flash",
	"gemini-2.5-flash-lite",
	"gemini-2.0-flash",
	"gemini-2.0-flash-lite",
	"gemma-3-27b-it",
}

// GRPC contains gRPC server configuration.
type GRPC struct {
	Addr string
}

// Generation contains model generation parameters.
type Generation struct {
	ModelCode    string
	Temperature  float32
	TopP         float32
	TopK         int32
	OutputLength int32
}

// Config holds the application configuration.
type Config struct {
	GRPC       GRPC
	Generation Generation
}

// Default returns a default configuration.
func Default() *Config {
	temp := float32(1.0)
	topP := float32(0.95)
	topK := int32(40)
	outputLength := int32(65536)

	return &Config{
		GRPC: GRPC{
			Addr: "localhost:50051",
		},
		Generation: Generation{
			ModelCode:    AvailableModels[0], // gemini-2.0-flash
			Temperature:  temp,
			TopP:         topP,
			TopK:         topK,
			OutputLength: outputLength,
		},
	}
}
