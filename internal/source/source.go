package source

import (
	"fmt"
	"os/exec"
)

// LoadProjectSource executes `git ls-files` and pipes it to `pcat` to get formatted source code
// of all tracked files in the current git repository.
func LoadProjectSource() (string, error) {
	cmd := exec.Command("sh", "-c", "git ls-files | rg -v '.+.md$' | pcat")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load project source: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
