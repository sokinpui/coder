package source

import (
	"fmt"
	"os/exec"
)

// LoadProjectSource executes `git ls-tree` and pipes it to `pcat` to get formatted source code.
func LoadProjectSource() (string, error) {
	cmd := exec.Command("sh", "-c", "git ls-tree --full-tree -r --name-only HEAD | pcat")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load project source: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
