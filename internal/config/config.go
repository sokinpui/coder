package config

import (
	"fmt"
	"github.com/sokinpui/coder/internal/utils"
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
	Addr string // e.g. "http://localhost:8080"
}

type Generation struct {
	ModelCode      string
	TitleModelCode string
	Temperature    float32
	TopP           float32
	TopK           float32
	OutputLength   int32
	StreamDelay    int
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

type TreeKeymap struct {
	Up       string
	Down     string
	Expand   string
	Collapse string
	Top      string
	Bottom   string
	Toggle   string
	Exit     string
}

type JumpKeymap struct {
	Up   string
	Down string
	Exit string
}

type Keymap struct {
	Submit      string
	Editor      string
	Paste       string
	History     string
	New         string
	Branch      string
	Finder      string
	Tree        string
	Search      string
	Jump        string
	ContextList string
	ApplyITF    string
	ScrollUp    string
	ScrollDown  string
	Suspend     string
	Visual      string

	VisualMode  VisualKeymap  `mapstructure:"visualmode"`
	HistoryView HistoryKeymap `mapstructure:"historyview"`
	TreeView    TreeKeymap    `mapstructure:"treeview"`
	JumpView    JumpKeymap    `mapstructure:"jumpview"`
}

type Config struct {
	Server          Server
	Generation      Generation
	Context         Context
	Clipboard       Clipboard
	UI              UI
	Keymap          Keymap
	AvailableModels []string `yaml:"-"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("server.addr", "http://localhost:8080")
	v.SetDefault("generation.modelcode", "gemini-3-flash-preview")
	v.SetDefault("generation.titlemodelcode", "gemini-2.5-flash-lite")
	v.SetDefault("generation.temperature", 0.0)
	v.SetDefault("generation.topp", 0.95)
	v.SetDefault("generation.topk", 0)
	v.SetDefault("generation.outputlength", 65536)
	v.SetDefault("generation.streamdelay", 0)
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
	v.SetDefault("keymap.tree", "ctrl+t")
	v.SetDefault("keymap.search", "ctrl+p")
	v.SetDefault("keymap.jump", "ctrl+q")
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

	v.SetDefault("keymap.treeview.up", "k")
	v.SetDefault("keymap.treeview.down", "j")
	v.SetDefault("keymap.treeview.expand", "l")
	v.SetDefault("keymap.treeview.collapse", "h")
	v.SetDefault("keymap.treeview.top", "g")
	v.SetDefault("keymap.treeview.bottom", "G")
	v.SetDefault("keymap.treeview.toggle", "space")
	v.SetDefault("keymap.treeview.exit", "q")

	v.SetDefault("keymap.jumpview.up", "k")
	v.SetDefault("keymap.jumpview.down", "j")
	v.SetDefault("keymap.jumpview.exit", "q")

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
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if !strings.HasPrefix(cfg.Server.Addr, "http") {
		cfg.Server.Addr = "http://" + cfg.Server.Addr
	}

	return &cfg, nil
}

func InitConfig(isLocal bool) error {
	var configDir string
	if isLocal {
		repoRoot, err := utils.FindRepoRoot()
		if err != nil {
			return fmt.Errorf("failed to find repo root: %w", err)
		}
		configDir = filepath.Join(repoRoot, ".coder")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config", "coder")
	}
	return ensureDirAndFile(configDir, "config.yaml", ConfigTemplate)
}

func ensureDirAndFile(dir, filename, content string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	return os.WriteFile(path, []byte(content), 0644)
}
