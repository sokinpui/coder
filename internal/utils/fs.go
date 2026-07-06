package utils

import (
	"fmt"
	"github.com/sokinpui/coder/pkg/sf"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func FindRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func GetProjectRoot() string {
	root, err := FindRepoRoot()
	if err != nil {
		cwd, _ := os.Getwd()
		return cwd
	}
	return root
}

func UserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

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

func SourceToFileList(dirs []string, initialFiles []string, exclusions []string) ([]string, error) {
	seen := make(map[string]struct{})
	allFiles := make([]string, 0)

	// Explicitly requested files bypass exclusions
	for _, f := range initialFiles {
		p := filepath.ToSlash(f)
		if _, ok := seen[p]; !ok {
			allFiles = append(allFiles, p)
			seen[p] = struct{}{}
		}
	}

	if len(dirs) > 0 {
		for _, f := range sf.Run(dirs, "file", exclusions, true) {
			p := filepath.ToSlash(f)
			if _, ok := seen[p]; !ok {
				allFiles = append(allFiles, p)
				seen[p] = struct{}{}
			}
		}
	}

	return allFiles, nil
}
