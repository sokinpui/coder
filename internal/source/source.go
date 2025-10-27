package source

import (
	"coder/internal/config"
	"coder/internal/utils"
	"fmt"
	"os/exec"
)

// LoadProjectSource executes `fd` and pipes it to `pcat` to get formatted source code
// of files in the current directory, respecting .gitignore.
func LoadProjectSource(sources *config.FileSources) (string, error) {
	if len(sources.Dirs) == 0 && len(sources.Files) == 0 {
		return "", nil
	}
	finalExclusions := append(Exclusions, sources.Exclusions...)
	allFiles, err := utils.SourceToFileList(sources.Dirs, sources.Files, finalExclusions)
	if err != nil {
		return "", err
	}

	pcatArgs := append([]string{"--no-header"}, allFiles...)
	cmd := exec.Command("pcat", pcatArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to load project source with pcat: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}
