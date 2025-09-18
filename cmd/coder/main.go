package main

import (
	"coder/internal/logger"
	"coder/internal/ui"
	"coder/internal/utils"
	"fmt"
	"os"
)

func main() {
	if _, err := utils.FindRepoRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: This application must be run from within a git repository.")
		os.Exit(1)
	}
	logger.Init()
	ui.Start()
}
