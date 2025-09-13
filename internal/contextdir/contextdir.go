package contextdir

import (
	"coder/internal/utils"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const contextDirName = "Context"

// LoadContext finds and reads all files from the context directory at the git repository root.
// It returns all file contents as provided documents.
// User-defined system instructions are not supported from the context directory.
func LoadContext() (systemInstructions string, providedDocuments string, err error) {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		// This should be caught by the initial check in main, but handle gracefully.
		return "", "", nil
	}

	contextPath := filepath.Join(repoRoot, contextDirName)

	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return "", "", nil
	}

	var documents []string

	walkErr := filepath.WalkDir(contextPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			contentBytes, readErr := os.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("failed to read file %s: %w", path, readErr)
			}

			content := string(contentBytes)

			// Make the path relative to the repo root for display
			relativePath, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				// Fallback to the full path if relative path fails
				relativePath = path
			}

			// Use filepath.ToSlash to ensure consistent path separators
			displayPath := filepath.ToSlash(relativePath)

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
		return "", "", nil
	}

	// systemInstructions is returned as empty because user-defined instructions are not loaded from files.
	return "", strings.Join(documents, "\n\n"), nil
}
