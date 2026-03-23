package config

import (
	"fmt"
	"github.com/sokinpui/coder/internal/utils"
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

// Clipboard contains configuration for custom copy/paste commands.
type Clipboard struct {
	CopyCmd  string
	PasteCmd string
}

// Server contains the API server configuration.
type Server struct {
	Addr string // e.g. "http://localhost:8080"
}

// Generation contains model generation parameters.
type Generation struct {
	ModelCode      string
	TitleModelCode string
	Temperature    float32
	TopP           float32
	TopK           float32
	OutputLength   int32
	StreamDelay    int
}

// UI contains configuration for the user interface.
type UI struct {
	MarkdownTheme string
}

// VisualKeymap contains keys for Visual Mode.
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

// HistoryKeymap contains keys for the History View.
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

// TreeKeymap contains keys for the File Tree.
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

// JumpKeymap contains keys for Jump Mode.
type JumpKeymap struct {
	Up   string
	Down string
	Exit string
}

// Keymap contains custom keybindings for global shortcuts.
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

// Config holds the application configuration.
type Config struct {
	Server          Server
	Generation      Generation
	Context         Context
	Clipboard       Clipboard
	UI              UI
	Keymap          Keymap
	AvailableModels []string `yaml:"-"`
}

// Load loads the application configuration from file and environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
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

	// Visual Mode defaults
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

	// History View defaults
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

	// Tree View defaults
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

	if !strings.HasPrefix(cfg.Server.Addr, "http") {
		cfg.Server.Addr = "http://" + cfg.Server.Addr
	}

	return &cfg, nil
}
