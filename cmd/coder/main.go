package main

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/logger"
	"github.com/sokinpui/coder/internal/ui"
	"github.com/sokinpui/coder/internal/utils"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var globalConfig bool

func main() {
	rootCmd := &cobra.Command{
		Use:   "coder",
		Short: "Coder is a TUI wrapper for LLM chat with code application shortcuts",
		Run: func(cmd *cobra.Command, args []string) {
			startApp("coding")
		},
	}

	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Start Coder in chat mode (no project context)",
		Run: func(cmd *cobra.Command, args []string) {
			startApp("chat")
		},
	}

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Edit the configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			editConfig()
		},
	}

	configCmd.Flags().BoolVarP(&globalConfig, "global", "g", false, "Edit the global configuration")
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(configCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
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

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil
	}

	return os.WriteFile(path, []byte(config.ConfigTemplate), 0644)
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

func startApp(mode string) {
	logger.Init()
	ui.Start(mode)
}
