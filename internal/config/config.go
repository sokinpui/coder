package config

import (
	"coder/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

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
)

var AvailableAppModes = []AppMode{CodingMode, DocumentingMode}

// FileSources specifies the files and directories to be included as project source.
type FileSources struct {
	Files      []string
	Dirs       []string
	Exclusions []string
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
	Sources    FileSources
}

// Load loads the application configuration from file and environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("appmode", string(CodingMode))
	v.SetDefault("grpc.addr", "localhost:50051")
	v.SetDefault("generation.modelcode", "gemini-2.5-pro")
	v.SetDefault("generation.temperature", 0.0)
	v.SetDefault("generation.topp", 0.95)
	v.SetDefault("generation.topk", 0)
	v.SetDefault("generation.outputlength", 65536)
	v.SetDefault("sources.dirs", []string{"."})
	v.SetDefault("sources.files", []string{})
	v.SetDefault("sources.exclusions", []string{})

	// Look for config in repo root .coder/
	repoRoot, err := utils.FindRepoRoot()
	if err == nil {
		v.AddConfigPath(filepath.Join(repoRoot, ".coder"))
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Also look in ~/.config/coder/
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "coder"))
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	// Set environment variable handling
	v.SetEnvPrefix("CODER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; ignore error and use defaults/env vars
	}

	// Unmarshal the config into our struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
