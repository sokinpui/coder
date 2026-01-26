package source

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/utils"
	"github.com/sokinpui/pcat"
	"fmt"
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

	if len(allFiles) == 0 {
		return "", nil
	}

	output, err := pcat.Run(
		allFiles, // specificFiles
		nil,      // directories
		nil,      // extensions
		nil,      // excludePatterns
		false,    // withLineNumbers
		true,     // hidden
		false,    // listOnly
	)
	if err != nil {
		return "", fmt.Errorf("failed to load project source with pcat: %w", err)
	}
	return output, nil
}
