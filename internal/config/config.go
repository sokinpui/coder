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
	Msg         string `mapstructure:"msg"`

	HistoryView HistoryKeymap `mapstructure:"historyview"`
}

type Config struct {
	Server          Server     `mapstructure:"server"`
	Generation      Generation `mapstructure:"generation"`
	Context         Context    `mapstructure:"context"`
	Clipboard       Clipboard  `mapstructure:"clipboard"`
	UI              UI         `mapstructure:"ui"`
	Keymap          Keymap     `mapstructure:"keymap"`
	AvailableModels []string                `yaml:"-"`
}

func DefaultConfig() Config {
	return Config{
		Server: Server{
			URL: "http://localhost:9001/v1",
		},
		Generation: Generation{
			ModelCode:       "aisrp/gemini-3-flash-preview",
			TitleModelCode:  "aisrp/gemini-flash-lite-latest",
			ReasoningEffort: "high",
		},
		Context: Context{
			Dirs:       []string{"."},
			Files:      []string{},
			Exclusions: []string{},
		},
		Clipboard: Clipboard{
			CopyCmd:  "",
			PasteCmd: "",
		},
		UI: UI{
			MarkdownTheme: "dark",
		},
		Keymap: Keymap{
			Submit:      "ctrl+j",
			Editor:      "ctrl+e",
			Paste:       "ctrl+v",
			History:     "ctrl+h",
			New:         "ctrl+n",
			Branch:      "ctrl+b",
			Finder:      "ctrl+f",
			ContextList: "ctrl+l",
			ApplyITF:    "ctrl+a",
			ScrollUp:    "ctrl+u",
			ScrollDown:  "ctrl+d",
			Suspend:     "ctrl+z",
			Msg:         "esc",
			HistoryView: HistoryKeymap{
				Up:           "k",
				Down:         "j",
				HalfPageUp:   "u",
				HalfPageDown: "d",
				Top:          "g",
				Bottom:       "G",
				Search:       "/",
				HistoryTab:   "h",
				ActiveTab:    "l",
				Exit:         "q",
			},
		},
	}
}

func DefaultTemplate() ([]byte, error) {
	data, err := yaml.Marshal(DefaultConfig())
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		sb.WriteString("# ")
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return []byte(sb.String()), nil
}

func Load() (*Config, error) {
	v := viper.New()

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

	cfg := DefaultConfig()
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
