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
	Files      []string `mapstructure:"files"`
	Dirs       []string `mapstructure:"dirs"`
	Exclusions []string `mapstructure:"exclusions"`
}

type Clipboard struct {
	CopyCmd  string `mapstructure:"copycmd"`
	PasteCmd string `mapstructure:"pastecmd"`
}

type Server struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"-" yaml:"-"`
}

type Generation struct {
	ModelCode       string `mapstructure:"modelcode"`
	TitleModelCode  string `mapstructure:"titlemodelcode"`
	ReasoningEffort string `mapstructure:"reasoningeffort"`
}

type UI struct {
	MarkdownTheme string `mapstructure:"markdowntheme"`
}

type VisualKeymap struct {
	Up         string `mapstructure:"up"`
	Down       string `mapstructure:"down"`
	Select     string `mapstructure:"select"`
	Swap       string `mapstructure:"swap"`
	Copy       string `mapstructure:"copy"`
	Delete     string `mapstructure:"delete"`
	Regenerate string `mapstructure:"regenerate"`
	Edit       string `mapstructure:"edit"`
	Branch     string `mapstructure:"branch"`
	New        string `mapstructure:"new"`
	Exit       string `mapstructure:"exit"`
}

type HistoryKeymap struct {
	Up           string `mapstructure:"up"`
	Down         string `mapstructure:"down"`
	HalfPageUp   string `mapstructure:"halfpageup"`
	HalfPageDown string `mapstructure:"halfpagedown"`
	Top          string `mapstructure:"top"`
	Bottom       string `mapstructure:"bottom"`
	Search       string `mapstructure:"search"`
	HistoryTab   string `mapstructure:"historytab"`
	ActiveTab    string `mapstructure:"activetab"`
	Exit         string `mapstructure:"exit"`
}

type Keymap struct {
	Submit      string `mapstructure:"submit"`
	Editor      string `mapstructure:"editor"`
	Paste       string `mapstructure:"paste"`
	History     string `mapstructure:"history"`
	New         string `mapstructure:"new"`
	Branch      string `mapstructure:"branch"`
	Finder      string `mapstructure:"finder"`
	ContextList string `mapstructure:"contextlist"`
	ApplyITF    string `mapstructure:"applyitf"`
	ScrollUp    string `mapstructure:"scrollup"`
	ScrollDown  string `mapstructure:"scrolldown"`
	Suspend     string `mapstructure:"suspend"`
	Visual      string `mapstructure:"visual"`

	VisualMode  VisualKeymap  `mapstructure:"visualmode"`
	HistoryView HistoryKeymap `mapstructure:"historyview"`
}

type ShellCommand struct {
	Description string `mapstructure:"description"`
	Exec        string `mapstructure:"exec"`
	CanAISee    bool   `mapstructure:"canAIsee"`
}

type Config struct {
	Server          Server                  `mapstructure:"server"`
	Generation      Generation              `mapstructure:"generation"`
	Context         Context                 `mapstructure:"context"`
	Clipboard       Clipboard               `mapstructure:"clipboard"`
	UI              UI                      `mapstructure:"ui"`
	Keymap          Keymap                  `mapstructure:"keymap"`
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
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if err == nil {
		configDir := filepath.Join(home, ".config", "coder")
		v.AddConfigPath(configDir)
		// Use MergeInConfig to avoid overwriting embedded defaults
		if err := v.MergeInConfig(); err != nil {
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
