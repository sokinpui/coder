package main

import (
	"coder/internal/logger"
	"coder/internal/ui"
	"coder/internal/utils"
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	tmpMode := flag.Bool("tmp", false, "Start in a temporary git repository")
	flag.Parse()

	if *tmpMode {
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
	} else {
		if _, err := utils.FindRepoRoot(); err != nil {
			fmt.Fprintln(os.Stderr, "Error: This application must be run from within a git repository.")
			os.Exit(1)
		}
	}
	logger.Init()
	ui.Start()
}
