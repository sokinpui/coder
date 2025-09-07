package main

import (
	"coder/internal/logger"
	"coder/internal/ui"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func isGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func main() {
	if !isGitRepository() {
		fmt.Fprintln(os.Stderr, "Error: This application must be run from within a git repository.")
		os.Exit(1)
	}
	logger.Init()
	ui.Start()
}
