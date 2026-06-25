package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/sokinpui/coder/internal/utils"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

//go:embed default.yaml
var defaultYAML []byte

type Context struct {
	Files      []string
	Dirs       []string
	Exclusions []string
}

type Clipboard struct {
	CopyCmd  string
	PasteCmd string
}

type Server struct {
	URL    string // e.g. "http://localhost:9001/v1/chat/completions"
	APIKey string `yaml:"-"`
}

type Generation struct {
	ModelCode       string
	TitleModelCode  string
	ReasoningEffort string
}

type UI struct {
	MarkdownTheme string
}

type VisualKeymap struct {
	Up         string
	Down       string
	Select     string
	Swap       string
	Copy       string
	Delete     string
	Regenerate string
	Edit       string
	Branch     string
	New        string
	Exit       string
}

type HistoryKeymap struct {
	Up           string
	Down         string
	HalfPageUp   string
	HalfPageDown string
	Top          string
	Bottom       string
	Search       string
	HistoryTab   string
	ActiveTab    string
	Exit         string
}

type Keymap struct {
	Submit      string
	Editor      string
	Paste       string
	History     string
	New         string
	Branch      string
	Finder      string
	ContextList string
	ApplyITF    string
	ScrollUp    string
	ScrollDown  string
	Suspend     string
	Visual      string

	VisualMode  VisualKeymap  `mapstructure:"visualmode"`
	HistoryView HistoryKeymap `mapstructure:"historyview"`
}

type ShellCommand struct {
	Description string
	Exec        string
	CanAISee    bool `mapstructure:"canAIsee"`
}

type Config struct {
	Server          Server
	Generation      Generation
	Context         Context
	Clipboard       Clipboard
	UI              UI
	Keymap          Keymap
	ShellCommands   map[string]ShellCommand `mapstructure:"shellcommands" yaml:"shellcommands"`
	AvailableModels []string                `yaml:"-"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(defaultYAML)); err != nil {
		return nil, fmt.Errorf("failed to read embedded default config: %w", err)
	}

	// Reset config path/name for file loading

	// Global config in ~/.config/coder/
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config", "coder")

		v.AddConfigPath(configDir)
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

	v.SetEnvPrefix("CODER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	cfg.Server.APIKey = os.Getenv("CODER_API_KEY")

	if !strings.HasPrefix(cfg.Server.URL, "http") {
		cfg.Server.URL = "http://" + cfg.Server.URL
	}

	return &cfg, nil
}

func UpdateLocalConfig(key string, value any) error {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("local config can only be updated within a git repository")
	}
	path := filepath.Join(repoRoot, ".coder", "config.yaml")

	m := make(map[string]any)
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &m)
	}

	parts := strings.Split(key, ".")
	curr := m
	for i := 0; i < len(parts)-1; i++ {
		next, ok := curr[parts[i]].(map[string]any)
		if !ok {
			next = make(map[string]any)
			curr[parts[i]] = next
		}
		curr = next
	}

	last := parts[len(parts)-1]
	if value == nil {
		delete(curr, last)
	} else {
		curr[last] = value
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, data, 0644)
}
