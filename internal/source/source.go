package source

import (
	"fmt"
	"github.com/sokinpui/pcat"
)

// LoadProjectSource executes `fd` and pipes it to `pcat` to get formatted source code
// of files in the current directory, respecting .gitignore.
func LoadProjectSource(files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	output, err := pcat.Run(
		files, // specificFiles
		nil,   // directories
		nil,   // extensions
		nil,   // excludePatterns
		false, // withLineNumbers
		true,  // hidden
		false, // listOnly
	)
	if err != nil {
		return "", fmt.Errorf("failed to load project source with pcat: %w", err)
	}
	return output, nil
}
