package utils

import (
	"fmt"
	"github.com/sokinpui/sf"
	"os"
	"os/exec"
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
	if len(dirs) == 0 {
		return initialFiles, nil
	}

	filesFromDirs := sf.Run(dirs, "file", exclusions, true)
	if len(filesFromDirs) == 0 {
		return initialFiles, nil
	}

	return append(initialFiles, filesFromDirs...), nil
}
