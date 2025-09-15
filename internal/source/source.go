package source

import (
	"coder/internal/config"
	"fmt"
	"os/exec"
	"strings"
)

// LoadProjectSource executes `fd` and pipes it to `pcat` to get formatted source code
// of files in the current directory, respecting .gitignore.
// It excludes common non-source files and directories.
func LoadProjectSource(mode config.AppMode) (string, error) {
	// Base exclusions for all modes.
	// These are typically not useful for AI context.
	exclusions := []string{
		"*-lock.json",
		"go.sum",
		".coder",
		".vscode",
		".idea",
		"dist",
		"bin",
		".env*",
		"*.log",
		"*.svg",
		"*.wasm",
	}

	// Exclude markdown files for non-documenting modes.
	if mode != config.DocumentingMode {
		exclusions = append(exclusions, "*.md")
	}

	var commandBuilder strings.Builder
	commandBuilder.WriteString("fd . --type=file")

	for _, exclusion := range exclusions {
		// Using single quotes to handle potential special characters in globs for the shell.
		commandBuilder.WriteString(fmt.Sprintf(" -E '%s'", exclusion))
	}

	commandBuilder.WriteString(" | pcat --no-header")
	command := commandBuilder.String()

	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load project source: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
