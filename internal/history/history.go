package history

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sokinpui/coder/internal/prompt"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	historyDirName = ".coder/history"
	indexFileName  = "index.json"
)

type Metadata struct {
	Title            string
	Mode             string
	CreatedAt        time.Time
	ModifiedAt       time.Time
	ContextFiles     []string
	ContextDocuments []string
	Exclusions       []string
	WorkingDir       string
}

type ConversationInfo struct {
	ID         string    `json:"id"`
	Filename   string    `json:"filename"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type ConversationData struct {
	Filename         string
	Title            string
	Mode             string
	CreatedAt        time.Time
	Messages         []types.Message
	ContextFiles     []string
	ContextDocuments []string
	Exclusions       []string
	WorkingDir       string
}

type IndexEntry struct {
	Filename   string    `json:"filename"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type Manager struct {
	historyPath string
	mu          sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()

	historyContent := BuildHistorySnippet(data.Messages)
	var contentBuilder strings.Builder
	contentBuilder.WriteString(prompt.ConversationHistoryHeader)
	contentBuilder.WriteString(historyContent)

	content := contentBuilder.String()
	now := time.Now()
	var fileBuf bytes.Buffer
	fmt.Fprintln(&fileBuf, "---")
	fmt.Fprintf(&fileBuf, "title: %s\n", data.Title)
	if data.Mode != "" {
		fmt.Fprintf(&fileBuf, "mode: %s\n", data.Mode)
	}
	fmt.Fprintf(&fileBuf, "createdAt: %s\n", data.CreatedAt.Format(time.RFC3339Nano))
	fmt.Fprintf(&fileBuf, "modifiedAt: %s\n", now.Format(time.RFC3339Nano))
	if data.WorkingDir != "" {
		fmt.Fprintf(&fileBuf, "workingDir: %s\n", data.WorkingDir)
	}
	writeYamlList(&fileBuf, "contextFiles", data.ContextFiles)
	writeYamlList(&fileBuf, "contextDocuments", data.ContextDocuments)
	writeYamlList(&fileBuf, "exclusions", data.Exclusions)
	fmt.Fprintln(&fileBuf, "---")
	fmt.Fprintln(&fileBuf, "")

	fileBuf.WriteString(content)

	filePath := filepath.Join(m.historyPath, data.Filename)
	if err := os.WriteFile(filePath, fileBuf.Bytes(), 0644); err != nil {
		return err
	}

	index := m.loadIndex()
	index[data.Filename] = IndexEntry{
		Filename:   data.Filename,
		Title:      data.Title,
		CreatedAt:  data.CreatedAt,
		ModifiedAt: now,
	}
	return m.saveIndex(index)
}

func writeYamlList(b *bytes.Buffer, key string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "%s:\n", key)
	for _, item := range items {
		fmt.Fprintf(b, "  - %s\n", item)
	}
}

var roleToMessageType = map[string]types.MessageType{
	"User:":                   types.UserMessage,
	"AI Assistant:":           types.AIMessage,
	"Image:":                  types.ImageMessage,
	"Command Execute:":        types.CommandMessage,
	"Command Execute Result:": types.CommandResultMessage,
	"Command Execute Error:":  types.CommandErrorResultMessage,
	"Instruction:":            types.InstructionMessage,
	"Source Code:":            types.SourceCodeMessage,
	"Shell Command:":          types.ShellCmdMessage,
	"Shell Command Result:":   types.ShellCmdResultMessage,
}

var imageMarkdownRegex = regexp.MustCompile(`^!\[image\]\((.*)\)$`)

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

func parseFrontmatter(scanner *bufio.Scanner) (*Metadata, bool) {
	metadata := &Metadata{}
	var currentKey string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			return metadata, true
		}

		if after, ok := strings.CutPrefix(line, "  - "); ok {
			val := strings.TrimSpace(after)
			switch currentKey {
			case "contextFiles", "files":
				metadata.ContextFiles = append(metadata.ContextFiles, val)
			case "contextDocuments", "documents":
				metadata.ContextDocuments = append(metadata.ContextDocuments, val)
			case "exclusions":
				metadata.Exclusions = append(metadata.Exclusions, val)
			}
			continue
		}

		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key, value := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		currentKey = key
		switch key {
		case "title":
			metadata.Title = value
		case "mode":
			metadata.Mode = value
		case "workingDir":
			metadata.WorkingDir = value
		case "contextFiles", "files":
			if value != "" {
				metadata.ContextFiles = append(metadata.ContextFiles, parseStringSlice(value)...)
			}
		case "contextDocuments", "documents":
			if value != "" {
				metadata.ContextDocuments = append(metadata.ContextDocuments, parseStringSlice(value)...)
			}
		case "exclusions":
			if value != "" {
				metadata.Exclusions = append(metadata.Exclusions, parseStringSlice(value)...)
			}
		case "createdAt", "modifiedAt":
			t, err := time.Parse(time.RFC3339Nano, value)
			if err != nil {
				continue
			}
			if key == "createdAt" {
				metadata.CreatedAt = t
			} else {
				metadata.ModifiedAt = t
			}
		}
	}
	return metadata, false
}

func ParseConversation(content []byte) (*Metadata, []types.Message, error) {
	parts := bytes.SplitN(content, []byte("---\n"), 3)
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("invalid file format: missing YAML frontmatter")
	}

	metaScanner := bufio.NewScanner(bytes.NewReader(parts[1]))
	metadata, _ := parseFrontmatter(metaScanner)

	bodyBytes := parts[2]
	historyHeader := []byte("# CONVERSATION HISTORY")
	_, after, ok := bytes.Cut(bodyBytes, historyHeader)
	if !ok {
		// No conversation history, but not an error.
		return metadata, []types.Message{}, nil
	}

	conversationContentBytes := after
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
					if currentMessage.Type != types.InstructionMessage && currentMessage.Type != types.SourceCodeMessage {
						messages = append(messages, *currentMessage)
					}
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
		if currentMessage.Type != types.InstructionMessage && currentMessage.Type != types.SourceCodeMessage {
			messages = append(messages, *currentMessage)
		}
	}

	return metadata, messages, nil
}

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

	metadata, closed := parseFrontmatter(scanner)

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if !closed {
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
	m.mu.Lock()
	defer m.mu.Unlock()

	dirEntries, err := os.ReadDir(m.historyPath)
	if err != nil {
		return nil, fmt.Errorf("could not read history directory: %w", err)
	}

	index := m.loadIndex()
	diskFiles := make(map[string]os.DirEntry)

	for _, entry := range dirEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		diskFiles[entry.Name()] = entry
	}

	indexDirty := false
	for filename := range index {
		if _, exists := diskFiles[filename]; !exists {
			delete(index, filename)
			indexDirty = true
		}
	}

	var toScan []os.DirEntry
	for name, entry := range diskFiles {
		cached, found := index[name]
		if !found {
			toScan = append(toScan, entry)
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(cached.ModifiedAt) {
			toScan = append(toScan, entry)
		}
	}

	if len(toScan) > 0 {
		scanned := m.scanEntriesParallel(toScan)
		for filename, entry := range scanned {
			index[filename] = entry
			indexDirty = true
		}
	}

	if indexDirty {
		_ = m.saveIndex(index)
	}

	conversations := make([]ConversationInfo, 0, len(index))
	for _, entry := range index {
		conversations = append(conversations, ConversationInfo{
			Filename:   entry.Filename,
			Title:      entry.Title,
			CreatedAt:  entry.CreatedAt,
			ModifiedAt: entry.ModifiedAt,
		})
	}

	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].ModifiedAt.After(conversations[j].ModifiedAt)
	})

	return conversations, nil
}

type scanResult struct {
	filename string
	entry    IndexEntry
}

func (m *Manager) scanEntriesParallel(entries []os.DirEntry) map[string]IndexEntry {
	workerCount := max(min(len(entries), runtime.NumCPU()*2), 1)

	jobs := make(chan os.DirEntry, len(entries))
	for _, entry := range entries {
		jobs <- entry
	}
	close(jobs)

	resultsChan := make(chan scanResult, len(entries))
	var wg sync.WaitGroup

	for range workerCount {
		wg.Go(func() {
			for entry := range jobs {
				item, ok := m.parseEntry(entry)
				if !ok {
					continue
				}
				resultsChan <- scanResult{filename: entry.Name(), entry: item}
			}
		})
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	results := make(map[string]IndexEntry, len(entries))
	for res := range resultsChan {
		results[res.filename] = res.entry
	}
	return results
}

func (m *Manager) parseEntry(entry os.DirEntry) (IndexEntry, bool) {
	filePath := filepath.Join(m.historyPath, entry.Name())
	metadata, err := ParseFileMetadata(filePath)
	if err != nil {
		return IndexEntry{}, false
	}

	fileInfo, infoErr := entry.Info()
	if metadata.CreatedAt.IsZero() && infoErr == nil {
		metadata.CreatedAt = fileInfo.ModTime()
	}
	if metadata.ModifiedAt.IsZero() {
		if !metadata.CreatedAt.IsZero() {
			metadata.ModifiedAt = metadata.CreatedAt
		} else if infoErr == nil {
			metadata.ModifiedAt = fileInfo.ModTime()
		}
	}

	return IndexEntry{
		Filename:   entry.Name(),
		Title:      metadata.Title,
		CreatedAt:  metadata.CreatedAt,
		ModifiedAt: metadata.ModifiedAt,
	}, true
}

func (m *Manager) loadIndex() map[string]IndexEntry {
	indexPath := filepath.Join(m.historyPath, indexFileName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return make(map[string]IndexEntry)
	}

	var entries map[string]IndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return make(map[string]IndexEntry)
	}
	return entries
}

func (m *Manager) saveIndex(entries map[string]IndexEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	indexPath := filepath.Join(m.historyPath, indexFileName)
	return os.WriteFile(indexPath, data, 0644)
}

func BuildHistorySnippet(messages []types.Message) string {
	var sb strings.Builder

	for i := range messages {
		msg := messages[i]

		if !msg.Type.IsHistory() {
			continue
		}

		switch msg.Type {
		case types.InstructionMessage:
			sb.WriteString("Instruction:\n")
			sb.WriteString(msg.Content)
		case types.SourceCodeMessage:
			sb.WriteString("Source Code:\n")
			sb.WriteString(msg.Content)
		case types.UserMessage:
			sb.WriteString("User:\n")
			sb.WriteString(msg.Content)
		case types.ImageMessage:
			sb.WriteString("Image:\n")
			fmt.Fprintf(&sb, "![image](%s)", msg.Content)
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
		case types.ShellCmdMessage:
			sb.WriteString("Shell Command:\n")
			sb.WriteString(msg.Content)
		case types.ShellCmdResultMessage:
			sb.WriteString("Shell Command Result:\n")
			sb.WriteString(msg.Content)
		}
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}
