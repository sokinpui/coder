package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/generation"
	"github.com/sokinpui/coder/internal/logger"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui"
	"github.com/sokinpui/coder/internal/utils"

	"github.com/spf13/cobra"
)

var (
	initialPrompt     string
	customInstruction string
	runModel          string
	chatMode          bool
	printContextFlag  bool
	configFlag        bool
	globalConfig      bool
	execMode          bool
	applyFlag         bool
	completionShell   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "coder [flags] [files...]",
		Short:   "Coder is a TUI-based AI code editor",
		Long:    "Coder is a TUI-based AI code editor that supports OpenAI-compatible services.",
		Version: utils.GetVersion(),
		Example: `  coder main.go
  coder -p "refactor this" main.go
  coder -e -p "explain this file" main.go
  coder --chat
  coder --context .
  coder --config -g`,
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
		Run: func(cmd *cobra.Command, args []string) {
			runCLI(cmd, args)
		},
	}

	rootCmd.Flags().StringVarP(&initialPrompt, "prompt", "p", "", "Prompt to start session with or execute in non-interactive mode")
	rootCmd.Flags().StringVarP(&customInstruction, "instruction", "i", "", "Custom system instruction to replace the default one")
	rootCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model to use for generation")
	rootCmd.Flags().BoolVarP(&chatMode, "chat", "c", false, "Start Coder in chat mode (no project context)")
	rootCmd.Flags().BoolVarP(&execMode, "exec", "e", false, "Execute a single AI request non-interactively and output to stdout")
	rootCmd.Flags().BoolVarP(&printContextFlag, "context", "C", false, "Print instructions and project context, then exit")
	rootCmd.Flags().BoolVar(&configFlag, "config", false, "Edit configuration file")
	rootCmd.Flags().BoolVarP(&globalConfig, "global", "g", false, "Use with --config to edit global configuration")
	rootCmd.Flags().BoolVarP(&applyFlag, "apply", "a", false, "Apply code changes using itf format from args or stdin")
	rootCmd.Flags().StringVar(&completionShell, "completion", "", "Generate autocompletion script (bash, zsh, fish, powershell)")

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCLI(cmd *cobra.Command, args []string) {
	if completionShell != "" {
		generateCompletion(cmd, completionShell)
		return
	}

	if configFlag {
		editConfig()
		return
	}

	if applyFlag {
		applyChanges(args)
		return
	}

	if printContextFlag {
		mode := session.ModeCoding
		if chatMode {
			mode = session.ModeChat
		}
		printContext(mode, args)
		return
	}

	if execMode {
		runSingleShot(args)
		return
	}

	if chatMode {
		startApp(session.ModeChat, initialPrompt, nil, customInstruction)
		return
	}

	files := collectFiles(args)
	startApp(session.ModeCoding, initialPrompt, files, customInstruction)
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

	content, err := config.DefaultTemplate()
	if err != nil {
		return fmt.Errorf("failed to generate template: %w", err)
	}

	return os.WriteFile(path, content, 0644)
}

func printContext(mode string, args []string) {
	files := collectFiles(args)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	allExclusions := append([]string{}, source.Exclusions...)
	allExclusions = append(allExclusions, cfg.Context.Exclusions...)

	var resolvedFiles []string
	if len(files) > 0 {
		resolvedFiles, _ = utils.SourceToFileList(nil, files, allExclusions)
	} else {
		resolvedFiles, _ = utils.SourceToFileList(cfg.Context.Dirs, cfg.Context.Files, allExclusions)
	}

	sess, err := session.New(cfg, mode, customInstruction, resolvedFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := sess.LoadContext(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var messages []types.Message
	if initialPrompt != "" {
		messages = append(messages, types.Message{Type: types.UserMessage, Content: initialPrompt})
	}

	fullPrompt := sess.BuildPrompt(messages)
	for _, msg := range fullPrompt {
		fmt.Printf("[%s]\n%s\n\n", msg.Type, msg.Content)
	}
}

func runSingleShot(args []string) {
	if initialPrompt == "" {
		fmt.Fprintln(os.Stderr, "Error: -p/--prompt is required for exec mode")
		os.Exit(1)
	}

	files := collectFiles(args)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	allExclusions := append([]string{}, source.Exclusions...)
	allExclusions = append(allExclusions, cfg.Context.Exclusions...)

	var resolvedFiles []string
	if len(files) > 0 {
		resolvedFiles, _ = utils.SourceToFileList(nil, files, allExclusions)
	} else {
		resolvedFiles, _ = utils.SourceToFileList(cfg.Context.Dirs, cfg.Context.Files, allExclusions)
	}

	sess, err := session.New(cfg, session.ModeCoding, customInstruction, resolvedFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}
	if err := sess.LoadContext(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading source: %v\n", err)
		os.Exit(1)
	}

	messages := []types.Message{
		{Type: types.UserMessage, Content: initialPrompt},
	}

	if runModel != "" {
		cfg.Generation.ModelCode = runModel
	}

	gen, err := generation.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing generator: %v\n", err)
		os.Exit(1)
	}

	promptMsgs := sess.BuildPrompt(messages)
	streamChan := make(chan types.StreamChunk, 100)
	ctx := context.Background()

	go gen.GenerateTask(ctx, promptMsgs, streamChan, nil)

	hasError := false
	for chunk := range streamChan {
		if strings.HasPrefix(chunk.Content, "Error:") {
			fmt.Fprintf(os.Stderr, "\n%s\n", chunk.Content)
			hasError = true
			continue
		}

		fmt.Print(chunk.Content)
	}

	fmt.Println()
	if hasError {
		os.Exit(1)
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

func applyChanges(args []string) {
	content := strings.Join(args, " ")
	if content == "" {
		content = readPipedInput()
	}

	if content == "" {
		fmt.Fprintln(os.Stderr, "Error: No content provided via arguments or stdin.")
		os.Exit(1)
	}

	res := commands.ExecuteItf(content, "")
	fmt.Println(res.Summary)
	if !res.Success {
		os.Exit(1)
	}
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
