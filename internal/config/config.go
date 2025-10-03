package config

var AvailableModels = []string{
	"gemini-2.5-pro",
	"gemini-2.5-flash-preview-09-2025",
	"gemini-2.5-flash",
	"gemini-2.5-flash-lite-preview-09-2025",
	"gemini-2.5-flash-lite",
	"gemini-2.0-flash",
	"gemini-2.0-flash-lite",
	"gemma-3-27b-it",
}

type AppMode string

const (
	CodingMode      AppMode = "Coding"
	DocumentingMode AppMode = "Documenting"
	AgentMode       AppMode = "Agent"
)

var AvailableAppModes = []AppMode{CodingMode, DocumentingMode, AgentMode}

type AgentName string

const (
	CodingAgent  AgentName = "coding_agent"
	WritingAgent AgentName = "writing_agent"
	GeneralAgent AgentName = "general_agent"
	MainAgent    AgentName = "main_agent"
)

// AgentGenerationConfig defines the generation settings for a specific agent.
type AgentGenerationConfig struct {
	ModelCode   string
	Temperature float32
	Tools       []string
}

// AgentConfigs holds the specific generation configurations for each agent.
var AgentConfigs = map[AgentName]AgentGenerationConfig{
	CodingAgent: {ModelCode: "gemini-2.5-pro", Temperature: 0.1, Tools: []string{
		"upsert_files",
	}},
	WritingAgent: {ModelCode: "gemini-2.5-pro", Temperature: 0.9, Tools: []string{
		"upsert_files",
	}},
	GeneralAgent: {ModelCode: "gemini-2.5-pro", Temperature: 0.9, Tools: []string{}},
	MainAgent: {ModelCode: "gemini-2.5-flash-preview-09-2025", Temperature: 0.9, Tools: []string{
		"coding_agent",
		"writing_agent",
		"general_agent",
		"read_files",
		"read_directories",
	}},
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
	TopK         float32
	OutputLength int32
}

// Config holds the application configuration.
type Config struct {
	AppMode    AppMode
	GRPC       GRPC
	Generation Generation
}

// Default returns a default configuration.
func Default() *Config {
	temp := float32(0)
	topP := float32(0.95)
	topK := float32(0) // disabled
	outputLength := int32(65536)

	return &Config{
		AppMode: CodingMode,
		GRPC: GRPC{
			Addr: "localhost:50051",
		},
		Generation: Generation{
			ModelCode:    "gemini-2.5-pro",
			Temperature:  temp,
			TopP:         topP,
			TopK:         topK,
			OutputLength: outputLength,
		},
	}
}
