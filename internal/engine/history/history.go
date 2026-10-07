package history

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

const historyDirName = ".coder/history"

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
	Mode       string    `json:"mode,omitempty"`
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

type Manager struct {
	historyPath string
	mu          sync.Mutex
}

func NewManager() (*Manager, error) {
	repoRoot := project.Root()
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
	contentBuilder.WriteString(ConversationHistoryHeader)
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
		Mode:       data.Mode,
		Title:      data.Title,
		CreatedAt:  data.CreatedAt,
		ModifiedAt: now,
	}
	return m.saveIndex(index)
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
	return m.ListConversationsByMode("")
}

func (m *Manager) ListConversationsByMode(modeFilter string) ([]ConversationInfo, error) {
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
		if !found || cached.Mode == "" {
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
		normMode := normalizeMode(entry.Mode)
		if modeFilter != "" && normMode != modeFilter {
			continue
		}
		conversations = append(conversations, ConversationInfo{
			Filename:   entry.Filename,
			Mode:       normMode,
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

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "chat":
		return "chat"
	case "agent":
		return "agent"
	default:
		return "coder"
	}
}
