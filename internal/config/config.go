package config

import (
	"coder/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Context specifies the files and directories to be included as project source.
type Context struct {
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
	ModelCode      string
	TitleModelCode string
	Temperature    float32
	TopP           float32
	TopK           float32
	OutputLength   int32
}

// Config holds the application configuration.
type Config struct {
	GRPC            GRPC
	Generation      Generation
	Context         Context
	AvailableModels []string `yaml:"-"`
}

// Load loads the application configuration from file and environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	v.SetDefault("grpc.addr", "localhost:50051")
	v.SetDefault("generation.modelcode", "gemini-2.5-pro")
	v.SetDefault("generation.titlemodelcode", "gemini-2.5-flash-lite-preview-09-2025")
	v.SetDefault("generation.temperature", 0.0)
	v.SetDefault("generation.topp", 0.95)
	v.SetDefault("generation.topk", 0)
	v.SetDefault("generation.outputlength", 65536)
	v.SetDefault("sources.dirs", []string{"."})
	v.SetDefault("sources.files", []string{})
	v.SetDefault("sources.exclusions", []string{})

	// Global config in ~/.config/coder/
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "coder"))
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read global config file: %w", err)
			}
		}
	}

	// Local config in repo root .coder/
	repoRoot, err := utils.FindRepoRoot()
	if err == nil {
		localViper := viper.New()
		localViper.AddConfigPath(filepath.Join(repoRoot, ".coder"))
		localViper.SetConfigName("config")
		localViper.SetConfigType("yaml")
		if err := localViper.ReadInConfig(); err == nil {
			if err := v.MergeConfigMap(localViper.AllSettings()); err != nil {
				return nil, fmt.Errorf("failed to merge local config: %w", err)
			}
		} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read local config file: %w", err)
		}
	}

	// Set environment variable handling
	v.SetEnvPrefix("CODER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal the config into our struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
