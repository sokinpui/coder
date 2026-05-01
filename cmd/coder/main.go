package main

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/logger"
	"github.com/sokinpui/coder/internal/modes"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui"
	"github.com/sokinpui/coder/internal/utils"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var (
	globalConfig      bool
	initialPrompt     string
	customInstruction string
	chatMode          bool
	configMode        bool
	contextMode       bool
	completionShell   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "coder [files...]",
		Version: uintVersion(),
		Short:   "Coder is a TUI wrapper for LLM chat with code application shortcuts",
		Run: func(cmd *cobra.Command, args []string) {
			if configMode {
				editConfig()
				return
			}

			if completionShell != "" {
				generateCompletion(cmd, completionShell)
				return
			}

			mode := "coding"
			if chatMode {
				mode = "chat"
			}

			if contextMode {
				printContext(mode, args)
				return
			}

			files := args
			if mode == "coding" {
				files = collectFiles(args)
			}

			startApp(mode, initialPrompt, files, customInstruction)
		},
	}

	rootCmd.Flags().BoolVar(&chatMode, "chat", false, "Start Coder in chat mode (no project context)")
	rootCmd.Flags().BoolVarP(&configMode, "config", "c", false, "Edit the configuration file")
	rootCmd.Flags().BoolVarP(&contextMode, "context", "C", false, "Print the instructions and project context")
	rootCmd.Flags().StringVar(&completionShell, "completion", "", "Generate autocompletion script for (bash, zsh, fish, powershell)")
	rootCmd.Flags().BoolVarP(&globalConfig, "global", "g", false, "Edit the global configuration (used with --config)")
	rootCmd.Flags().StringVarP(&initialPrompt, "prompt", "p", "", "Initial prompt to start the session with")
	rootCmd.Flags().StringVarP(&customInstruction, "instruction", "i", "", "Custom system instruction to replace the default one")

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func uintVersion() string {
	return utils.GetVersion()
}

func generateCompletion(cmd *cobra.Command, shell string) {
	var err error
	switch shell {
	case "bash":
		err = cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		err = cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		err = cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", shell)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating completion: %v\n", err)
		os.Exit(1)
	}
}

func editConfig() {
	configPath, err := getConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := ensureConfigFile(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	runEditor(configPath)
}

func getConfigPath() (string, error) {
	if globalConfig {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		return filepath.Join(home, ".config", "coder", "config.yaml"), nil
	}

	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		return "", fmt.Errorf("local config can only be edited from within a git repository. Use --global to edit the global config")
	}
	return filepath.Join(repoRoot, ".coder", "config.yaml"), nil
}

func ensureConfigFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	_, err := os.Stat(path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	return os.WriteFile(path, []byte(config.ConfigTemplate), 0644)
}

func printContext(mode string, args []string) {
	files := collectFiles(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(files) > 0 {
		cfg.Context.Dirs = []string{}
		cfg.Context.Files = []string{}
		cfg.Context.Exclusions = []string{}
		for _, p := range files {
			info, err := os.Stat(p)
			if err != nil {
				continue
			}
			if info.IsDir() {
				cfg.Context.Dirs = append(cfg.Context.Dirs, p)
			} else {
				cfg.Context.Files = append(cfg.Context.Files, p)
			}
		}
	}

	strategy := modes.GetStrategy(mode, customInstruction)
	if err := strategy.LoadSourceCode(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var messages []types.Message
	if initialPrompt != "" {
		messages = append(messages, types.Message{Type: types.UserMessage, Content: initialPrompt})
	}

	fullPrompt := strategy.BuildPrompt(messages)
	for _, msg := range fullPrompt {
		fmt.Printf("[%s]\n%s\n\n", msg.Type, msg.Content)
	}
}

func runEditor(path string) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to run editor: %v\n", err)
		os.Exit(1)
	}
}

func startApp(mode string, prompt string, contextFiles []string, instruction string) {
	logger.Init()
	ui.Start(mode, prompt, contextFiles, instruction)
}

func collectFiles(args []string) []string {
	var files []string

	// Expand globs for positional arguments
	for _, arg := range args {
		matches, err := filepath.Glob(arg)
		if err != nil || len(matches) == 0 {
			// If not a glob or no matches, treat as a literal path
			files = append(files, arg)
			continue
		}
		files = append(files, matches...)
	}

	// Handle piped input if it looks like a file list
	if piped := readPipedInput(); piped != "" && isFileList(piped) {
		lines := strings.SplitSeq(strings.TrimSpace(piped), "\n")
		for line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			// Avoid duplicates if already added via positional args
			if slices.Contains(files, trimmed) {
				continue
			}
			files = append(files, trimmed)
		}
	}
	return files
}

func readPipedInput() string {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return ""
	}

	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}

	return string(bytes)
}

func isFileList(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}
	lines := strings.SplitSeq(trimmed, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := os.Stat(line); err != nil {
			return false
		}
	}
	return true
}
