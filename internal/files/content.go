package files

import (
	"coder/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetFileContent reads and returns the content of a file within the repo.
func GetFileContent(relativePath string) (string, error) {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		return "", err
	}

	// Security: clean the path and ensure it doesn't try to escape the repo root.
	cleanPath := filepath.Clean(relativePath)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("invalid file path: %s", relativePath)
	}

	fullPath := filepath.Join(repoRoot, cleanPath)

	// Security: ensure the final path is still within the repo root.
	if !strings.HasPrefix(fullPath, repoRoot) {
		return "", fmt.Errorf("invalid file path, escapes repository root: %s", relativePath)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("could not read file %s: %w", relativePath, err)
	}

	return string(content), nil
}
