package history

import (
	"bytes"
	"coder/internal/core"
	"coder/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	historyDirName = ".coder/history"
)

// Metadata holds the parsed YAML frontmatter from a history file.
type Metadata struct {
	Title      string
	CreatedAt  time.Time
	ModifiedAt time.Time
}

// ConversationInfo holds metadata for a single conversation.
type ConversationInfo struct {
	Filename   string    `json:"filename"`
	Title      string    `json:"title"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// ConversationData holds all the necessary information to save a conversation history file.
type ConversationData struct {
	Filename           string
	Title              string
	CreatedAt          time.Time
	Messages           []core.Message
	Role               string
	SystemInstructions string
	RelatedDocuments   string
	ProjectSourceCode  string
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

var roleToMessageType = map[string]core.MessageType{
	"User:":                   core.UserMessage,
	"AI Assistant:":           core.AIMessage,
	"Action Execute:":         core.ActionMessage,
	"Action Execute Result:":  core.ActionResultMessage,
	"Action Execute Error:":   core.ActionErrorResultMessage,
	"Command Execute:":        core.CommandMessage,
	"Command Execute Result:": core.CommandResultMessage,
	"Command Execute Error:":  core.CommandErrorResultMessage,
}

// ParseConversation parses the content of a history file into its metadata and messages.
func ParseConversation(content []byte) (*Metadata, []core.Message, error) {
	parts := bytes.SplitN(content, []byte("---\n"), 3)
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("invalid file format: missing YAML frontmatter")
	}

	metadata := &Metadata{}
	metaLines := strings.Split(string(parts[1]), "\n")
	for _, line := range metaLines {
		if line == "" {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch key {
		case "title":
			metadata.Title = value
		case "createdAt":
			t, err := time.Parse(time.RFC3339Nano, value)
			if err == nil {
				metadata.CreatedAt = t
			}
		case "modifiedAt":
			t, err := time.Parse(time.RFC3339Nano, value)
			if err == nil {
				metadata.ModifiedAt = t
			}
		}
	}

	body := string(parts[2])
	historyHeader := "# CONVERSATION HISTORY"
	historyIndex := strings.Index(body, historyHeader)
	if historyIndex == -1 {
		// No conversation history, but not an error.
		return metadata, []core.Message{}, nil
	}

	conversationContent := body[historyIndex+len(historyHeader):]
	conversationContent = strings.TrimSpace(conversationContent)

	var messages []core.Message
	var currentMessage *core.Message
	var contentBuilder strings.Builder

	lines := strings.Split(conversationContent, "\n")

	for _, line := range lines {
		foundRole := false
		for role, msgType := range roleToMessageType {
			if strings.HasPrefix(line, role) {
				if currentMessage != nil {
					currentMessage.Content = strings.TrimSpace(contentBuilder.String())
					messages = append(messages, *currentMessage)
				}
				contentBuilder.Reset()
				currentMessage = &core.Message{Type: msgType}
				contentBuilder.WriteString(strings.TrimSpace(strings.TrimPrefix(line, role)))
				foundRole = true
				break
			}
		}
		if !foundRole && currentMessage != nil {
			contentBuilder.WriteString("\n")
			contentBuilder.WriteString(line)
		}
	}

	if currentMessage != nil {
		currentMessage.Content = strings.TrimSpace(contentBuilder.String())
		messages = append(messages, *currentMessage)
	}

	return metadata, messages, nil
}

// LoadConversation reads a history file from disk and parses it.
func (m *Manager) LoadConversation(filename string) (*Metadata, []core.Message, error) {
	filePath := filepath.Join(m.historyPath, filename)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not stat history file %s: %w", filename, err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read history file %s: %w", filename, err)
	}

	metadata, messages, err := ParseConversation(content)
	if err != nil {
		return nil, nil, err
	}

	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = fileInfo.ModTime()
	}

	return metadata, messages, nil
}

// ListConversations scans the history directory and returns info for each conversation.
func (m *Manager) ListConversations() ([]ConversationInfo, error) {
	files, err := os.ReadDir(m.historyPath)
	if err != nil {
		return nil, fmt.Errorf("could not read history directory: %w", err)
	}

	var conversations []ConversationInfo
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(m.historyPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			// Silently skip files that can't be read
			continue
		}

		metadata, _, err := ParseConversation(content)
		if err != nil {
			// Silently skip files that can't be parsed
			continue
		}

		conversations = append(conversations, ConversationInfo{
			Filename:   file.Name(),
			Title:      metadata.Title,
			ModifiedAt: metadata.ModifiedAt,
		})
	}

	// Sort by modified date, newest first.
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].ModifiedAt.After(conversations[j].ModifiedAt)
	})

	return conversations, nil
}
