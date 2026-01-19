package main

import (
	"coder/internal/config"
	"coder/internal/logger"
	"coder/internal/ui"
	"coder/internal/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var tmpMode bool
var globalConfig bool

func main() {
	rootCmd := &cobra.Command{
		Use:   "coder",
		Short: "Coder is a TUI wrapper for LLM chat with code application shortcuts",
		Run: func(cmd *cobra.Command, args []string) {
			startApp()
		},
	}

	rootCmd.Flags().BoolVar(&tmpMode, "tmp", false, "Start in a temporary git repository")

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Edit the configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			editConfig()
		},
	}

	configCmd.Flags().BoolVarP(&globalConfig, "global", "g", false, "Edit the global configuration")
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

func startApp() {
	if tmpMode {
		setupTempEnvironment()
	} else {
		validateWorkingDir()
	}
	logger.Init()
	ui.Start()
}

func setupTempEnvironment() {
	dir, err := os.MkdirTemp("", "coder-tmp-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create temporary directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chdir(dir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to change to temporary directory: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command("git", "init")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize git repository: %v\nOutput: %s\n", err, string(output))
		os.Exit(1)
	}
	fmt.Printf("Started in temporary git repository: %s\n", dir)
	fmt.Println("This directory will not be automatically cleaned up.")
}

func validateWorkingDir() {
	if _, err := utils.FindRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: This application must be run from within a git repository.")
		os.Exit(1)
	}
}
