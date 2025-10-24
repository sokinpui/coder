package utils

import (
	"bufio"
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

func UserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
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

// SourceToFileList constructs a final list of files from given directories and an initial file list.
func SourceToFileList(dirs []string, initialFiles []string, exclusions []string) ([]string, error) {
	var filesFromDirs []string
	if len(dirs) > 0 {
		var quotedDirs []string
		for _, d := range dirs {
			// quoting for directory paths with spaces or special characters.
			quotedDirs = append(quotedDirs, fmt.Sprintf("'%s'", strings.ReplaceAll(d, "'", "'\\''")))
		}

		var commandBuilder strings.Builder
		commandBuilder.WriteString(fmt.Sprintf("fd . %s --type=file --hidden", strings.Join(quotedDirs, " ")))

		for _, exclusion := range exclusions {
			commandBuilder.WriteString(fmt.Sprintf(" -E '%s'", exclusion))
		}

		command := commandBuilder.String()

		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.Output()
		if err != nil {
			// Re-run to get combined output for better error message
			cmdForErr := exec.Command("sh", "-c", command)
			combinedOutput, _ := cmdForErr.CombinedOutput()
			return nil, fmt.Errorf("failed to list files with fd: %w\nOutput: %s", err, string(combinedOutput))
		}

		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			if line := scanner.Text(); line != "" {
				filesFromDirs = append(filesFromDirs, line)
			}
		}
	}

	allFiles := append(initialFiles, filesFromDirs...)
	return allFiles, nil
}
