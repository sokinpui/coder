package source

import (
	"coder/internal/config"
	"fmt"
	"os/exec"
)

// LoadProjectSource executes `git ls-files` and pipes it to `pcat` to get formatted source code
// of all tracked files in the current git repository.
func LoadProjectSource(mode config.AppMode) (string, error) {
	var command string
	if mode == config.DocumentingMode {
		// Include markdown files for documenting mode
		command = "git ls-files | pcat --no-header"
	} else {
		// Exclude markdown files for other modes (e.g., Coding)
		command = "git ls-files | rg -v '.+.md$' | pcat --no-header"
	}

	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load project source: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
