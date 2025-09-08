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

func (m *Manager) filterMessages(messages []core.Message) []core.Message {
	var filteredMessages []core.Message
	for i := 0; i < len(messages); i++ {
		msg := messages[i]

		// Check for command that resulted in an error
		if msg.Type == core.UserMessage && strings.HasPrefix(msg.Content, "/") {
			if i+1 < len(messages) {
				nextMsg := messages[i+1]
				if nextMsg.Type == core.CommandErrorResultMessage {
					i++ // Skip both the command and the error message
					continue
				}
			}
		}
		filteredMessages = append(filteredMessages, msg)
	}
	return filteredMessages
}

// hasMeaningfulContent checks if the conversation has more than just the initial message.
func hasMeaningfulContent(messages []core.Message) bool {
	for _, msg := range messages {
		if msg.Type == core.UserMessage || msg.Type == core.AIMessage {
			return true
		}
	}
	return false
}

// SaveConversation saves the conversation to a markdown file.
// It filters out commands that resulted in errors.
func (m *Manager) SaveConversation(messages []core.Message, systemInstructions, providedDocuments string) error {
	filteredMessages := m.filterMessages(messages)

	if !hasMeaningfulContent(filteredMessages) {
		return nil // Nothing to save
	}

	content := core.BuildPrompt(systemInstructions, providedDocuments, filteredMessages)

	// The prompt builder adds a trailing "AI Assistant:\n" which we don't want in the saved file.
	content = strings.TrimSuffix(content, "AI Assistant:\n")
	content = strings.TrimSpace(content)

	if content == "" {
		return nil
	}

	timestamp := time.Now().Format("15-04-02-01-2006") // hour-minutes-day-month-year
	fileName := fmt.Sprintf("%s.md", timestamp)
	filePath := filepath.Join(m.historyPath, fileName)

	return os.WriteFile(filePath, []byte(content), 0644)
}
