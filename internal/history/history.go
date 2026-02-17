package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sokinpui/coder/internal/modes"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	historyDirName = ".coder/history"
)

type Metadata struct {
	Title      string
	CreatedAt  time.Time
	ModifiedAt time.Time
	Files      []string
	Dirs       []string
	Exclusions []string
	WorkingDir string
}

type ConversationInfo struct {
	Filename   string    `json:"filename"`
	Title      string    `json:"title"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type ConversationData struct {
	Filename   string
	Title      string
	CreatedAt  time.Time
	Messages   []types.Message
	Context    string // Role Instruction + source code
	Files      []string
	Dirs       []string
	Exclusions []string
	WorkingDir string
}

type Manager struct {
	historyPath string
}

func NewManager() (*Manager, error) {
	repoRoot := utils.GetProjectRoot()
	historyPath := filepath.Join(repoRoot, historyDirName)
	if err := os.MkdirAll(historyPath, 0755); err != nil {
		return nil, fmt.Errorf("could not create history directory at %s: %w", historyPath, err)
	}

	return &Manager{historyPath: historyPath}, nil
}

func (m *Manager) SaveConversation(data *ConversationData) error {
	historyContent := BuildHistorySnippet(data.Messages)
	trimmedHeaders := strings.TrimSpace(data.Context)

	var contentBuilder strings.Builder
	if trimmedHeaders != "" {
		contentBuilder.WriteString(trimmedHeaders)
	}

	if historyContent != "" {
		contentBuilder.WriteString(modes.Separator)
		contentBuilder.WriteString(modes.ConversationHistoryHeader)
		contentBuilder.WriteString(historyContent)
	}

	content := contentBuilder.String()
	var fileBuf bytes.Buffer
	fmt.Fprintln(&fileBuf, "---")
	fmt.Fprintf(&fileBuf, "title: %s\n", data.Title)
	fmt.Fprintf(&fileBuf, "createdAt: %s\n", data.CreatedAt.Format(time.RFC3339Nano))
	fmt.Fprintf(&fileBuf, "modifiedAt: %s\n", time.Now().Format(time.RFC3339Nano))
	if data.WorkingDir != "" {
		fmt.Fprintf(&fileBuf, "workingDir: %s\n", data.WorkingDir)
	}
	if len(data.Files) > 0 {
		paths, err := json.Marshal(data.Files)
		if err == nil {
			fmt.Fprintf(&fileBuf, "files: %s\n", string(paths))
		}
	}
	if len(data.Dirs) > 0 {
		dirs, err := json.Marshal(data.Dirs)
		if err == nil {
			fmt.Fprintf(&fileBuf, "dirs: %s\n", string(dirs))
		}
	}
	if len(data.Exclusions) > 0 {
		exclusions, err := json.Marshal(data.Exclusions)
		if err == nil {
			fmt.Fprintf(&fileBuf, "exclusions: %s\n", string(exclusions))
		}
	}
	fmt.Fprintln(&fileBuf, "---")
	fmt.Fprintln(&fileBuf, "")

	fileBuf.WriteString(content)

	filePath := filepath.Join(m.historyPath, data.Filename)
	return os.WriteFile(filePath, fileBuf.Bytes(), 0644)
}

var roleToMessageType = map[string]types.MessageType{
	"User:":                   types.UserMessage,
	"AI Assistant:":           types.AIMessage,
	"Image:":                  types.ImageMessage,
	"Command Execute:":        types.CommandMessage,
	"Command Execute Result:": types.CommandResultMessage,
	"Command Execute Error:":  types.CommandErrorResultMessage,
}

var imageMarkdownRegex = regexp.MustCompile(`^!\[image\]\((.*)\)$`)

// processMessageContent trims whitespace and handles special message content parsing,
// like extracting the path from a markdown image link for ImageMessages.
func processMessageContent(msg *types.Message, rawContent string) {
	content := strings.TrimSpace(rawContent)
	if msg.Type == types.ImageMessage {
		matches := imageMarkdownRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			content = matches[1]
		}
	}
	msg.Content = content
}

func parseStringSlice(value string) []string {
	var s []string
	if err := json.Unmarshal([]byte(value), &s); err != nil {
		return nil
	}
	return s
}

func ParseConversation(content []byte) (*Metadata, []types.Message, error) {
	parts := bytes.SplitN(content, []byte("---\n"), 3)
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("invalid file format: missing YAML frontmatter")
	}

	metadata := &Metadata{}
	metaScanner := bufio.NewScanner(bytes.NewReader(parts[1]))
	for metaScanner.Scan() {
		line := metaScanner.Text()
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
		case "files":
			metadata.Files = parseStringSlice(value)
		case "dirs":
			metadata.Dirs = parseStringSlice(value)
		case "exclusions":
			metadata.Exclusions = parseStringSlice(value)
		case "workingDir":
			metadata.WorkingDir = value
		}
	}

	bodyBytes := parts[2]
	historyHeader := []byte("# CONVERSATION HISTORY")
	historyIndex := bytes.Index(bodyBytes, historyHeader)
	if historyIndex == -1 {
		// No conversation history, but not an error.
		return metadata, []types.Message{}, nil
	}

	conversationContentBytes := bodyBytes[historyIndex+len(historyHeader):]
	conversationContentBytes = bytes.TrimSpace(conversationContentBytes)

	var messages []types.Message
	var currentMessage *types.Message
	var contentBuilder strings.Builder

	convScanner := bufio.NewScanner(bytes.NewReader(conversationContentBytes))
	for convScanner.Scan() {
		line := convScanner.Text()
		foundRole := false
		for role, msgType := range roleToMessageType {
			if strings.HasPrefix(line, role) {
				if currentMessage != nil {
					processMessageContent(currentMessage, contentBuilder.String())
					messages = append(messages, *currentMessage)
				}
				contentBuilder.Reset()
				currentMessage = &types.Message{Type: msgType}
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
		processMessageContent(currentMessage, contentBuilder.String())
		messages = append(messages, *currentMessage)
	}

	return metadata, messages, nil
}

// ParseFileMetadata reads the YAML frontmatter from a history file to get its metadata.
func ParseFileMetadata(filePath string) (*Metadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Expect "---"
	if !scanner.Scan() || scanner.Text() != "---" {
		return nil, fmt.Errorf("invalid file format: missing YAML frontmatter start")
	}

	metadata := &Metadata{}
	inFrontmatter := true
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			inFrontmatter = false
			break
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
		case "files":
			metadata.Files = parseStringSlice(value)
		case "dirs":
			metadata.Dirs = parseStringSlice(value)
		case "exclusions":
			metadata.Exclusions = parseStringSlice(value)
		case "workingDir":
			metadata.WorkingDir = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if inFrontmatter {
		return nil, fmt.Errorf("invalid file format: YAML frontmatter not closed")
	}

	return metadata, nil
}

func (m *Manager) LoadConversation(filename string) (*Metadata, []types.Message, error) {
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
		metadata, err := ParseFileMetadata(filePath)
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

// BuildHistorySnippet constructs a string representation of a list of messages for copying.
func BuildHistorySnippet(messages []types.Message) string {
	var sb strings.Builder

	for i := range messages {
		msg := messages[i]

		switch msg.Type {
		case types.UserMessage:
			sb.WriteString("User:\n")
			sb.WriteString(msg.Content)
		case types.ImageMessage:
			sb.WriteString("Image:\n")
			sb.WriteString(fmt.Sprintf("![image](%s)", msg.Content))
		case types.AIMessage:
			if msg.Content == "" {
				continue
			}
			sb.WriteString("AI Assistant:\n")
			sb.WriteString(msg.Content)
		case types.CommandMessage:
			sb.WriteString("Command Execute:\n")
			sb.WriteString(msg.Content)
		case types.CommandResultMessage:
			sb.WriteString("Command Execute Result:\n")
			sb.WriteString(msg.Content)
		case types.CommandErrorResultMessage:
			sb.WriteString("Command Execute Error:\n")
			sb.WriteString(msg.Content)
		default:
			// Skip system messages like InitMessage, DirectoryMessage
			continue
		}
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}
