package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/factory"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/server"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui"
	"github.com/sokinpui/coder/pkg/version"

	"github.com/spf13/cobra"
)

var (
	initialPrompt     string
	customInstruction string
	runModel          string
	runProtocol       string
	chatMode          bool
	agentMode         bool
	coderMode         bool
	configFlag        bool
	globalConfig      bool
	execMode          bool
	completionShell   string
	headlessMode      bool
	serverPort        int
	serverSocket      string
	wsMode            bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "coder [flags] [files... or prompt]",
		Short:   "Coder is a terminal AI coding agent and code editor",
		Long:    "Coder is a TUI-based AI code editor and autonomous terminal coding agent.",
		Version: version.Get(),
		Example: `  coder main.go
  coder -p "refactor this" main.go
  coder -e -p "explain this file" main.go
  coder --chat
  coder --agent "Fix the failing test in pkg/itf"
  coder -a -m gpt-4o
  coder --config -g
  coder --headless`,
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
	rootCmd.Flags().StringVarP(&runProtocol, "protocol", "P", "responses", "Protocol to use: chat or responses")
	rootCmd.Flags().BoolVarP(&chatMode, "chat", "c", false, "Start Coder in chat mode (no project context)")
	rootCmd.Flags().BoolVarP(&agentMode, "agent", "a", false, "Start Coder in agent mode (autonomous AI tool-calling agent)")
	rootCmd.Flags().BoolVar(&coderMode, "coder", false, "Start Coder in coder mode (default)")
	rootCmd.Flags().BoolVarP(&execMode, "exec", "e", false, "Execute a single AI request non-interactively and output to stdout")
	rootCmd.Flags().BoolVar(&configFlag, "config", false, "Edit configuration file")
	rootCmd.Flags().BoolVarP(&globalConfig, "global", "g", false, "Use with --config to edit global configuration")
	rootCmd.Flags().StringVar(&completionShell, "completion", "", "Generate autocompletion script (bash, zsh, fish, powershell)")
	rootCmd.Flags().BoolVar(&headlessMode, "headless", false, "Run as headless JSON-RPC server over stdio")
	rootCmd.Flags().IntVar(&serverPort, "port", 0, "Run headless server listening on TCP port")
	rootCmd.Flags().StringVar(&serverSocket, "socket", "", "Run headless server listening on Unix socket path")
	rootCmd.Flags().BoolVar(&wsMode, "ws", false, "Run headless server with WebSocket/HTTP support")

	_ = rootCmd.RegisterFlagCompletionFunc("completion", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"bash", "zsh", "fish", "powershell"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = rootCmd.RegisterFlagCompletionFunc("protocol", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"chat", "responses"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if cmd.Flags().Changed("protocol") && runProtocol != "" {
		cfg.Server.Protocol = runProtocol
	}
	if runModel != "" {
		cfg.Coder.ModelCode = runModel
		cfg.Agent.ModelCode = runModel
	}

	srv := server.New(cfg)
	if wsMode {
		port := serverPort
		if port == 0 {
			port = 9005
		}
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen on port %d: %v\n", port, err)
			os.Exit(1)
		}
		fmt.Printf("Coder WebSocket server listening on ws://localhost:%d/ws\n", port)
		_ = srv.ServeHTTP(listener)
		return
	}
	if serverPort > 0 {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen on port %d: %v\n", serverPort, err)
			os.Exit(1)
		}
		_ = srv.ServeListener(listener)
		return
	}
	if serverSocket != "" {
		listener, err := net.Listen("unix", serverSocket)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen on socket %s: %v\n", serverSocket, err)
			os.Exit(1)
		}
		_ = srv.ServeListener(listener)
		return
	}
	_ = srv.ServeStdio()
}

func runCLI(cmd *cobra.Command, args []string) {
	if headlessMode || serverPort > 0 || serverSocket != "" || wsMode {
		runServer(cmd)
		return
	}

	if completionShell != "" {
		generateCompletion(cmd, completionShell)
		return
	}

	if configFlag {
		editConfig()
		return
	}

	mode := engine.ModeCoder
	if agentMode {
		mode = engine.ModeAgent
	} else if chatMode {
		mode = engine.ModeChat
	}

	prompt := initialPrompt
	if mode == engine.ModeAgent || mode == engine.ModeChat {
		if len(args) > 0 {
			posPrompt := strings.Join(args, " ")
			if prompt != "" {
				prompt = prompt + "\n" + posPrompt
			} else {
				prompt = posPrompt
			}
		}
	}

	if execMode {
		runSingleShot(cmd, mode, prompt, args)
		return
	}

	var files []string
	if mode == engine.ModeCoder {
		files = collectFiles(args)
	}
	startApp(cmd, mode, prompt, files, customInstruction)
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

	repoRoot, err := project.FindRepoRoot()
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

func runSingleShot(cmd *cobra.Command, mode string, prompt string, args []string) {
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "Error: -p/--prompt or positional prompt is required for exec mode")
		os.Exit(1)
	}

	var files []string
	if mode == engine.ModeCoder {
		files = collectFiles(args)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cmd.Flags().Changed("protocol") && runProtocol != "" {
		cfg.Server.Protocol = runProtocol
	}
	if runModel != "" {
		cfg.Coder.ModelCode = runModel
		cfg.Agent.ModelCode = runModel
	}

	sess, err := factory.NewSession(cfg, mode, customInstruction, files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}
	if ctxCtrl, ok := sess.(engine.ContextController); ok {
		if err := ctxCtrl.LoadContext(); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading source: %v\n", err)
			os.Exit(1)
		}
	}

	eventChan, err := sess.Submit(context.Background(), prompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting request: %v\n", err)
		os.Exit(1)
	}

	hasError := false
	for ev := range eventChan {
		if ev.Kind == types.EventError {
			fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			hasError = true
			continue
		}
		if ev.Kind == types.EventChunk && ev.Content != "" {
			fmt.Print(ev.Content)
		}
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

func startApp(cmd *cobra.Command, mode string, prompt string, contextFiles []string, instruction string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	if cmd.Flags().Changed("protocol") && runProtocol != "" {
		cfg.Server.Protocol = runProtocol
	}
	if runModel != "" {
		cfg.Coder.ModelCode = runModel
		cfg.Agent.ModelCode = runModel
	}

	sess, err := factory.NewSession(cfg, mode, instruction, contextFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}
	_ = ui.Start(sess, prompt)
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
