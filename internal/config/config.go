package config

import (
	"fmt"
	"github.com/sokinpui/coder/internal/utils"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

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

	v.SetDefault("server.url", "http://localhost:9001/v1")
	v.SetDefault("generation.modelcode", "gemini-3-flash-preview")
	v.SetDefault("generation.titlemodelcode", "gemini-2.5-flash-lite")
	v.SetDefault("generation.reasoningeffort", "high")
	v.SetDefault("context.dirs", []string{"."})
	v.SetDefault("context.files", []string{})
	v.SetDefault("context.exclusions", []string{})
	v.SetDefault("clipboard.copycmd", "")
	v.SetDefault("clipboard.pastecmd", "")
	v.SetDefault("ui.markdowntheme", "dark")
	v.SetDefault("keymap.submit", "ctrl+j")
	v.SetDefault("keymap.editor", "ctrl+e")
	v.SetDefault("keymap.paste", "ctrl+v")
	v.SetDefault("keymap.history", "ctrl+h")
	v.SetDefault("keymap.new", "ctrl+n")
	v.SetDefault("keymap.branch", "ctrl+b")
	v.SetDefault("keymap.finder", "ctrl+f")
	v.SetDefault("keymap.contextlist", "ctrl+l")
	v.SetDefault("keymap.applyitf", "ctrl+a")
	v.SetDefault("keymap.scrollup", "ctrl+u")
	v.SetDefault("keymap.scrolldown", "ctrl+d")
	v.SetDefault("keymap.suspend", "ctrl+z")
	v.SetDefault("keymap.visual", "esc")

	v.SetDefault("keymap.visualmode.up", "k")
	v.SetDefault("keymap.visualmode.down", "j")
	v.SetDefault("keymap.visualmode.select", "v")
	v.SetDefault("keymap.visualmode.swap", "o")
	v.SetDefault("keymap.visualmode.copy", "y")
	v.SetDefault("keymap.visualmode.delete", "d")
	v.SetDefault("keymap.visualmode.regenerate", "g")
	v.SetDefault("keymap.visualmode.edit", "e")
	v.SetDefault("keymap.visualmode.branch", "b")
	v.SetDefault("keymap.visualmode.new", "n")
	v.SetDefault("keymap.visualmode.exit", "i")

	v.SetDefault("keymap.historyview.up", "k")
	v.SetDefault("keymap.historyview.down", "j")
	v.SetDefault("keymap.historyview.halfpageup", "u")
	v.SetDefault("keymap.historyview.halfpagedown", "d")
	v.SetDefault("keymap.historyview.top", "g")
	v.SetDefault("keymap.historyview.bottom", "G")
	v.SetDefault("keymap.historyview.search", "/")
	v.SetDefault("keymap.historyview.historytab", "h")
	v.SetDefault("keymap.historyview.activetab", "l")
	v.SetDefault("keymap.historyview.exit", "q")

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
