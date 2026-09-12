package project

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func IsGitRepo() bool {
	cmd := exec.Command("git", "status")
	return cmd.Run() == nil
}

func FindRepoRoot() (string, error) {
	if !IsGitRepo() {
		return "", fmt.Errorf("not a git repository")
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func Root() string {
	root, err := FindRepoRoot()
	if err != nil {
		cwd, _ := os.Getwd()
		return cwd
	}
	return root
}

func DirInfo() string {
	var dirInfoParts []string
	if cwd, err := os.Getwd(); err == nil {
		dirInfoParts = append(dirInfoParts, fmt.Sprintf("Current directory: %s", shortenPath(cwd)))
	}
	if repoRoot, err := FindRepoRoot(); err == nil {
		dirInfoParts = append(dirInfoParts, fmt.Sprintf("Project Root: %s", shortenPath(repoRoot)))
	}
	return strings.Join(dirInfoParts, "\n")
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
