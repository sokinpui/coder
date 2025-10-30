package source

import (
	"coder/internal/config"
	"coder/internal/utils"
	"fmt"
	"os/exec"
)

// LoadProjectSource executes `fd` and pipes it to `pcat` to get formatted source code
// of files in the current directory, respecting .gitignore.
func LoadProjectSource(context *config.Context) (string, error) {
	if len(context.Dirs) == 0 && len(context.Files) == 0 {
		return "", nil
	}
	finalExclusions := append(Exclusions, context.Exclusions...)
	allFiles, err := utils.SourceToFileList(context.Dirs, context.Files, finalExclusions)
	if err != nil {
		return "", err
	}

	pcatArgs := append([]string{}, allFiles...)
	cmd := exec.Command("pcat", pcatArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load project source with pcat: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
