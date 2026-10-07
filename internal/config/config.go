package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sokinpui/coder/internal/project"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Context struct {
	Files      []string `mapstructure:"files" yaml:"files"`
	Dirs       []string `mapstructure:"dirs" yaml:"dirs"`
	Exclusions []string `mapstructure:"exclusions" yaml:"exclusions"`
}

type Clipboard struct {
	CopyCmd  string `mapstructure:"copycmd"`
	PasteCmd string `mapstructure:"pastecmd"`
}

type Server struct {
	URL      string `mapstructure:"url"`
	Protocol string `mapstructure:"protocol"`
	APIKey   string `mapstructure:"-" yaml:"-"`
}

type Title struct {
	ModelCode string `mapstructure:"modelcode" yaml:"modelcode"`
}

type ModelConfig struct {
	ModelCode       string `mapstructure:"modelcode" yaml:"modelcode"`
	ReasoningEffort string `mapstructure:"reasoningeffort" yaml:"reasoningeffort"`
}

type Coder struct {
	ModelCode       string  `mapstructure:"modelcode" yaml:"modelcode"`
	ReasoningEffort string  `mapstructure:"reasoningeffort" yaml:"reasoningeffort"`
	Context         Context `mapstructure:"context" yaml:"context"`
	Keymap          Keymap  `mapstructure:"keymap" yaml:"keymap"`
}

func (c Coder) ModelConfig() ModelConfig {
	return ModelConfig{
		ModelCode:       c.ModelCode,
		ReasoningEffort: c.ReasoningEffort,
	}
}

type Agent struct {
	ModelCode       string         `mapstructure:"modelcode" yaml:"modelcode"`
	SecondaryModel  string         `mapstructure:"secondarymodel" yaml:"secondarymodel,omitempty"`
	ReasoningEffort string         `mapstructure:"reasoningeffort" yaml:"reasoningeffort"`
	MaxIterations   int            `mapstructure:"max_iterations" yaml:"max_iterations"`
	Keymap          Keymap         `mapstructure:"keymap" yaml:"keymap"`
	Permission      map[string]any `mapstructure:"permission" yaml:"permission,omitempty"`
	Tools           map[string]any `mapstructure:"tools" yaml:"tools,omitempty"`
}

func (a Agent) ModelConfig() ModelConfig {
	return ModelConfig{
		ModelCode:       a.ModelCode,
		ReasoningEffort: a.ReasoningEffort,
	}
}

func (a Agent) SecondaryModelConfig() ModelConfig {
	if a.SecondaryModel != "" {
		return ModelConfig{
			ModelCode:       a.SecondaryModel,
			ReasoningEffort: a.ReasoningEffort,
		}
	}
	return a.ModelConfig()
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
	AddFile     string `mapstructure:"addfile"`
	ContextList string `mapstructure:"contextlist"`
	ApplyITF    string `mapstructure:"applyitf"`
	ScrollUp    string `mapstructure:"scrollup"`
	ScrollDown  string `mapstructure:"scrolldown"`
	Suspend     string `mapstructure:"suspend"`
	Msg         string `mapstructure:"msg"`

	HistoryView HistoryKeymap `mapstructure:"historyview"`
}

type Config struct {
	Server          Server         `mapstructure:"server" yaml:"server"`
	Title           Title          `mapstructure:"title" yaml:"title"`
	Coder           Coder          `mapstructure:"coder" yaml:"coder"`
	Agent           Agent          `mapstructure:"agent" yaml:"agent"`
	Clipboard       Clipboard      `mapstructure:"clipboard" yaml:"clipboard"`
	Tools           map[string]any `mapstructure:"tools" yaml:"tools,omitempty"`
	AvailableModels []string       `mapstructure:"-" yaml:"-"`
}

func DefaultKeymap() Keymap {
	return Keymap{
		Submit:      "ctrl+j",
		Editor:      "ctrl+e",
		Paste:       "ctrl+v",
		History:     "ctrl+h",
		New:         "ctrl+n",
		Branch:      "ctrl+b",
		Finder:      "ctrl+f",
		AddFile:     "ctrl+t",
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
	}
}

func DefaultConfig() Config {
	return Config{
		Server: Server{
			URL:      "http://localhost:9001/v1",
			Protocol: "responses",
		},
		Title: Title{
			ModelCode: "aisrp/gemini-flash-lite-latest",
		},
		Coder: Coder{
			ModelCode:       "aisrp/gemini-flash-latest",
			ReasoningEffort: "high",
			Context: Context{
				Files:      []string{},
				Dirs:       []string{"."},
				Exclusions: []string{},
			},
			Keymap: DefaultKeymap(),
		},
		Agent: Agent{
			ModelCode:       "aisrp/gemini-flash-lite-latest",
			ReasoningEffort: "high",
			MaxIterations:   50,
			Keymap:          DefaultKeymap(),
			Tools:           map[string]any{},
		},
		Clipboard: Clipboard{
			CopyCmd:  "",
			PasteCmd: "",
		},
	}
}

func DefaultTemplate() ([]byte, error) {
	data, err := yaml.Marshal(DefaultConfig())
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	for line := range strings.SplitSeq(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.HasPrefix(line, "title:") ||
			strings.HasPrefix(line, "coder:") ||
			strings.HasPrefix(line, "agent:") ||
			strings.HasPrefix(line, "clipboard:") {
			sb.WriteByte('\n')
		}
		sb.WriteString("# ")
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return []byte(sb.String()), nil
}

func Load() (*Config, error) {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))

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
	repoRoot, err := project.FindRepoRoot()
	if err == nil {
		localViper := viper.NewWithOptions(viper.KeyDelimiter("::"))
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

	if v.IsSet("generation.modelcode") && !v.IsSet("coder.modelcode") {
		cfg.Coder.ModelCode = v.GetString("generation.modelcode")
	}
	if v.IsSet("generation.titlemodelcode") && !v.IsSet("title.modelcode") {
		cfg.Title.ModelCode = v.GetString("generation.titlemodelcode")
	}
	if v.IsSet("generation.reasoningeffort") && !v.IsSet("coder.reasoningeffort") {
		cfg.Coder.ReasoningEffort = v.GetString("generation.reasoningeffort")
	}
	if v.IsSet("agent.secondary_model") && !v.IsSet("agent.secondarymodel") {
		cfg.Agent.SecondaryModel = v.GetString("agent.secondary_model")
	}
	if v.IsSet("context.files") && !v.IsSet("coder.context.files") {
		cfg.Coder.Context.Files = v.GetStringSlice("context.files")
	}
	if v.IsSet("context.dirs") && !v.IsSet("coder.context.dirs") {
		cfg.Coder.Context.Dirs = v.GetStringSlice("context.dirs")
	}
	if v.IsSet("context.exclusions") && !v.IsSet("coder.context.exclusions") {
		cfg.Coder.Context.Exclusions = v.GetStringSlice("context.exclusions")
	}
	if v.IsSet("tools") && !v.IsSet("agent.tools") {
		cfg.Agent.Tools = v.GetStringMap("tools")
	}
	if len(cfg.Agent.Tools) == 0 && len(cfg.Tools) > 0 {
		cfg.Agent.Tools = cfg.Tools
	}

	if cfg.Agent.MaxIterations <= 0 {
		cfg.Agent.MaxIterations = 50
	}
	cfg.Server.APIKey = os.Getenv("CODER_API_KEY")

	if !strings.HasPrefix(cfg.Server.URL, "http") {
		cfg.Server.URL = "http://" + cfg.Server.URL
	}

	return &cfg, nil
}
