package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FindRepoRoot finds the root directory of the current git repository.
func FindRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ShortenPath replaces the user's home directory with ~ in a given path.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// GetDirInfoContent returns a formatted string with the current directory and project root.
func GetDirInfoContent() string {
	var dirInfoParts []string
	if cwd, err := os.Getwd(); err == nil {
		dirInfoParts = append(dirInfoParts, fmt.Sprintf("Current directory: %s", ShortenPath(cwd)))
	}
	if repoRoot, err := FindRepoRoot(); err == nil {
		dirInfoParts = append(dirInfoParts, fmt.Sprintf("Project Root: %s", ShortenPath(repoRoot)))
	}
	return strings.Join(dirInfoParts, "\n")
}
