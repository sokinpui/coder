package contextdir

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const contextDirName = "Context"
const systemInstructionsFile = "SystemInstructions.md"

// LoadContext finds and reads all files from the context directory.
// It separates the content of `SystemInstructions.md` from other documents.
// If multiple `SystemInstructions.md` files are found, the last one encountered wins.
func LoadContext() (systemInstructions string, providedDocuments string, err error) {
	if _, err := os.Stat(contextDirName); os.IsNotExist(err) {
		return "", "", nil
	}

	var documents []string
	var sysInstructions string

	walkErr := filepath.WalkDir(contextDirName, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			contentBytes, readErr := os.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("failed to read file %s: %w", path, readErr)
			}

			if d.Name() == systemInstructionsFile {
				sysInstructions = string(contentBytes)
				return nil // Don't add to the regular documents list
			}

			content := string(contentBytes)

			// Use filepath.ToSlash to ensure consistent path separators
			displayPath := filepath.ToSlash(path)

			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}

			docString := fmt.Sprintf("`%s`\n```\n%s```", displayPath, content)
			documents = append(documents, docString)
		}
		return nil
	})

	if walkErr != nil {
		return "", "", fmt.Errorf("error walking context directory: %w", walkErr)
	}

	if len(documents) == 0 {
		return sysInstructions, "", nil
	}

	return sysInstructions, strings.Join(documents, "\n\n"), nil
}
