package itf

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

type CLIConfig struct {
	OutputDiffFix bool
	Undo          bool
	Redo          bool
	NoAnimation   bool
	Extensions    []string
	Completion    string
	Files         []string
}

var cfg = &CLIConfig{}

var rootCmd = &cobra.Command{
	Use:   "itf",
	Version: getVersion(),
	Short: "Parse content from stdin or clipboard to update files.",
	Long: `Parse content from stdin (pipe) or clipboard to update files in Neovim.

Example: pbpaste | itf -e py`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Completion != "" {
			return handleCompletion(cmd)
		}

		if cfg.Undo && cfg.Redo {
			return fmt.Errorf("error: --undo and --redo are mutually exclusive")
		}

		normalizeExtensions()

		itfCfg := &Config{
			OutputDiffFix: cfg.OutputDiffFix,
			Undo:          cfg.Undo,
			Redo:          cfg.Redo,
			Extensions:    cfg.Extensions,
			Files:         cfg.Files,
		}

		app, err := NewApp(itfCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize application: %w", err)
		}

		if cfg.OutputDiffFix {
			_, err := app.Execute()
			return err
		}

		ui := NewTUI(app, cfg.NoAnimation)
		return ui.Run()
	},
}

func handleCompletion(cmd *cobra.Command) error {
	switch cfg.Completion {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return fmt.Errorf("unsupported shell for completion: %s", cfg.Completion)
	}
}

func normalizeExtensions() {
	for i, ext := range cfg.Extensions {
		if len(ext) > 0 && ext[0] != '.' {
			cfg.Extensions[i] = "." + ext
		}
	}
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "devel"
	}

	return info.Main.Version
}

func init() {
	rootCmd.Flags().StringVar(&cfg.Completion, "completion", "", "Generate completion script")
	rootCmd.Flags().BoolVarP(&cfg.OutputDiffFix, "output-diff-fix", "o", false, "Print corrected diff")
	rootCmd.Flags().BoolVar(&cfg.NoAnimation, "no-animation", false, "Disable spinner")
	rootCmd.Flags().StringSliceVarP(&cfg.Extensions, "extension", "e", []string{}, "Filter by extension")
	rootCmd.Flags().StringSliceVarP(&cfg.Files, "file", "f", []string{}, "Filter by files")
	rootCmd.Flags().BoolVarP(&cfg.Undo, "undo", "u", false, "Undo last op")
	rootCmd.Flags().BoolVarP(&cfg.Redo, "redo", "r", false, "Redo last op")

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
}

func Execute() error {
	return rootCmd.Execute()
}
