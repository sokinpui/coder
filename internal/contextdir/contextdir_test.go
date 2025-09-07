package contextdir

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// This test file acts as a command-line tool to test context directory loading.
// It does not run any standard tests but uses the `TestMain` entry point
// to process a command-line argument.
//
// Usage:
// go test ./internal/contextdir -v -- /path/to/your/directory
//
// The `--` separates arguments for `go test` from arguments for the test program.

func TestMain(m *testing.M) {
	// Find the user-provided arguments after the `--` separator.
	args := os.Args[1:]
	var userArgs []string
	for i, arg := range args {
		if arg == "--" {
			if len(args) > i+1 {
				userArgs = args[i+1:]
			}
			break
		}
	}

	if len(userArgs) != 1 {
		fmt.Println("Usage: go test ./internal/contextdir -v -- /path/to/directory")
		os.Exit(1)
	}

	targetDir := userArgs[0]

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		fmt.Printf("Error: Provided path does not exist: %s\n", targetDir)
		os.Exit(1)
	}

	docs, files, err := LoadDocumentsAndListFiles(targetDir)
	if err != nil {
		fmt.Printf("Error loading documents from %s: %v\n", targetDir, err)
		os.Exit(1)
	}

	fmt.Println("\n--- PROVIDED DOCUMENTS ---")
	if docs == "" {
		fmt.Println("(empty)")
	} else {
		fmt.Println(docs)
	}

	fmt.Println("\n--- FILES PROCESSED IN ORDER ---")
	if len(files) == 0 {
		fmt.Println("(no files found)")
	} else {
		fmt.Println(strings.Join(files, "\n"))
	}
	fmt.Println()

	// Exit without running any other tests.
	os.Exit(0)
}
