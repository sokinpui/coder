package history

import (
	"bytes"
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

// ConversationData holds all the necessary information to save a conversation history file.
type ConversationData struct {
	Filename          string
	Title             string
	CreatedAt         time.Time
	Messages          []core.Message
	Role              string
	SystemInstructions string
	RelatedDocuments  string
	ProjectSourceCode string
}

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
// It includes YAML frontmatter for metadata.
func (m *Manager) SaveConversation(data *ConversationData) error {
	content := core.BuildPrompt(data.Role, data.SystemInstructions, data.RelatedDocuments, data.ProjectSourceCode, data.Messages)

	// The prompt builder adds a trailing "AI Assistant:\n" which we don't want in the saved file.
	content = strings.TrimSuffix(content, "AI Assistant:\n")
	content = strings.TrimSpace(content)

	if content == "" && data.Title == "New Chat" {
		return nil
	}

	var fileBuf bytes.Buffer
	fmt.Fprintln(&fileBuf, "---")
	fmt.Fprintf(&fileBuf, "title: %s\n", data.Title)
	fmt.Fprintf(&fileBuf, "createdAt: %s\n", data.CreatedAt.Format(time.RFC3339Nano))
	fmt.Fprintf(&fileBuf, "modifiedAt: %s\n", time.Now().Format(time.RFC3339Nano))
	fmt.Fprintln(&fileBuf, "---")
	fmt.Fprintln(&fileBuf, "") // newline after metadata block

	fileBuf.WriteString(content)

	filePath := filepath.Join(m.historyPath, data.Filename)
	return os.WriteFile(filePath, fileBuf.Bytes(), 0644)
}
