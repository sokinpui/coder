package history

import (
	"coder/internal/core"
	"coder/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	historyDirName = ".coder/history"
)

// Manager handles saving conversation history.
type Manager struct {
	historyPath string
}

// NewManager creates a new history manager.
// It finds the git repository root and ensures the history directory exists.
func NewManager() (*Manager, error) {
	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("could not find git repository root: %w", err)
	}

	historyPath := filepath.Join(repoRoot, historyDirName)
	if err := os.MkdirAll(historyPath, 0755); err != nil {
		return nil, fmt.Errorf("could not create history directory at %s: %w", historyPath, err)
	}

	return &Manager{historyPath: historyPath}, nil
}

// SaveConversation saves the conversation to a markdown file.
// It use BuildPrompt to construct the conversation history.
func (m *Manager) SaveConversation(messages []core.Message, systemInstructions, providedDocuments, projectSourceCode string) error {

	content := core.BuildPrompt(systemInstructions, providedDocuments, projectSourceCode, messages)

	// The prompt builder adds a trailing "AI Assistant:\n" which we don't want in the saved file.
	content = strings.TrimSuffix(content, "AI Assistant:\n")
	content = strings.TrimSpace(content)

	if content == "" {
		return nil
	}

	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%d.md", timestamp)
	filePath := filepath.Join(m.historyPath, fileName)

	return os.WriteFile(filePath, []byte(content), 0644)
}
