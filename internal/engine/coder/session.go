package coder

import (
	"context"
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/generation"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Session struct {
	ID                 string
	config             *config.Config
	Runtime            *Runtime
	historyManager     *history.Manager
	messages           []types.Message
	cancelGeneration   context.CancelFunc
	title              string
	titleGenerated     bool
	historyFilename    string
	createdAt          time.Time
	mode               string
	instruction        string
	projectSourceFiles []string
	lastModifiedFiles  []string
	hasAppliedChanges  bool
	contextFiles       []string
	contextDocuments   []string
	documentMessages   []types.Message
	cachedDocMessages  map[string][]types.Message
	docModTimes        map[string]time.Time
	contextLoadedAt    time.Time
	isStreaming        bool
}

func New(cfg *config.Config, mode string, instruction string, contextFiles []string) (*Session, error) {
	if mode == "" {
		mode = engine.ModeCoder
	}
	return NewWithMessages(cfg, nil, mode, instruction, contextFiles)
}

func NewWithMessages(cfg *config.Config, initialMessages []types.Message, mode string, instruction string, contextFiles []string) (*Session, error) {
	if mode == "" {
		mode = engine.ModeCoder
	}
	hist, err := history.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history manager: %w", err)
	}

	messages := make([]types.Message, len(initialMessages))
	copy(messages, initialMessages)

	cfgCopy := *cfg
	cfgCopy.Coder.Context.Files = append([]string{}, cfg.Coder.Context.Files...)
	cfgCopy.Coder.Context.Dirs = append([]string{}, cfg.Coder.Context.Dirs...)
	cfgCopy.Coder.Context.Exclusions = append([]string{}, cfg.Coder.Context.Exclusions...)

	rt, err := NewRuntime(&cfgCopy)
	if err != nil {
		return nil, err
	}

	allExclusions := append([]string{}, source.Exclusions...)
	allExclusions = append(allExclusions, cfgCopy.Coder.Context.Exclusions...)

	var resolvedContextFiles []string
	var resolvedContextDocs []string
	switch mode {
	case engine.ModeCoder:
		initialPaths := contextFiles
		if len(initialPaths) == 0 {
			initialPaths = append(append([]string{}, getSafeContextDirs(cfgCopy.Coder.Context.Dirs)...), cfgCopy.Coder.Context.Files...)
		}
		resolvedContextFiles, resolvedContextDocs, _ = source.Add(nil, nil, initialPaths, allExclusions)
	default:
		if len(contextFiles) > 0 {
			resolvedContextFiles, resolvedContextDocs, _ = source.Add(nil, nil, contextFiles, allExclusions)
		}
	}

	s := &Session{
		ID:                fmt.Sprintf("%d", time.Now().UnixNano()),
		config:            &cfgCopy,
		Runtime:           rt,
		historyManager:    hist,
		messages:          messages,
		title:             "New Chat",
		titleGenerated:    false,
		createdAt:         time.Now(),
		historyFilename:   "",
		mode:              mode,
		instruction:       instruction,
		contextFiles:      resolvedContextFiles,
		contextDocuments:  resolvedContextDocs,
		cachedDocMessages: make(map[string][]types.Message),
		docModTimes:       make(map[string]time.Time),
	}

	return s, nil
}

func getSafeContextDirs(configuredDirs []string) []string {
	if project.IsGitRepo() {
		return configuredDirs
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return configuredDirs
	}

	cwd, err := os.Getwd()
	if err != nil {
		return configuredDirs
	}

	if filepath.Clean(cwd) == filepath.Clean(home) {
		return nil
	}
	return configuredDirs
}

func (s *Session) GetConfig() *config.Config {
	return s.config
}

func (s *Session) ReloadConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	s.config = cfg
	s.Runtime.Config = cfg.Coder.ModelConfig()
	s.Runtime.Generator.Config = cfg.Coder.ModelConfig()
	s.Runtime.Generator.TitleModel = cfg.Title.ModelCode
	s.Runtime.Generator.BaseURL = cfg.Server.URL
	s.Runtime.Generator.Protocol = cfg.Server.Protocol
	s.Runtime.Generator.APIKey = cfg.Server.APIKey
	return nil
}

func (s *Session) GetGenerator() *generation.Generator {
	if s.Runtime == nil {
		return nil
	}
	return s.Runtime.Generator
}

func (s *Session) GetHistoryManager() *history.Manager {
	return s.historyManager
}

func (s *Session) GetCreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) GetInstruction() string {
	return s.instruction
}

func (s *Session) GetContextFiles() []string {
	return s.contextFiles
}

func (s *Session) GetContextDocuments() []string {
	return s.contextDocuments
}

func (s *Session) SetContextDocuments(docs []string) {
	s.contextDocuments = docs
}

func (s *Session) GetDocumentPageCount(doc string) int {
	doc = filepath.ToSlash(doc)
	if msgs, ok := s.cachedDocMessages[doc]; ok {
		return len(msgs)
	}
	return 0
}

func (s *Session) GetDocumentMessages() []types.Message {
	return s.documentMessages
}

func (s *Session) SetContextFiles(files []string) {
	s.contextFiles = files
	s.contextLoadedAt = time.Time{}
}

func AppendUnique(original []string, newItems []string) []string {
	return source.AppendUnique(original, newItems)
}

func (s *Session) GetLastModifiedFiles() []string {
	return s.lastModifiedFiles
}

func (s *Session) SetLastModifiedFiles(files []string) {
	s.lastModifiedFiles = files
}

func (s *Session) HasAppliedChanges() bool {
	return s.hasAppliedChanges
}

func (s *Session) SetHasAppliedChanges(applied bool) {
	s.hasAppliedChanges = applied
}

func (s *Session) GetMode() string {
	if s.mode == "" {
		return engine.ModeCoder
	}
	return s.mode
}

func (s *Session) SetModel(model string) {
	s.config.Coder.ModelCode = model
	if s.Runtime != nil {
		s.Runtime.Config.ModelCode = model
		if s.Runtime.Generator != nil {
			s.Runtime.Generator.Config.ModelCode = model
		}
	}
}

func (s *Session) HasChatHistory() bool {
	if s.historyFilename != "" {
		return true
	}
	for _, msg := range s.messages {
		switch msg.Type {
		case types.UserMessage, types.AIMessage, types.ImageMessage, types.ToolCallMessage, types.ToolResultMessage:
			if msg.Type == types.AIMessage && strings.TrimSpace(msg.Content) == "" {
				continue
			}
			return true
		}
	}
	return false
}

func (s *Session) ReadAgentFiles(input string, paths []string) (commands.CommandOutput, bool) {
	return commands.CommandOutput{}, false
}

func (s *Session) RespondToolConfirmation(callID string, response types.ToolConfirmResponse) error {
	return fmt.Errorf("tool confirmation not supported in coder session")
}

func (s *Session) SetMode(mode string) error {
	norm := engine.NormalizeMode(mode)
	if s.HasChatHistory() && s.mode != norm {
		return fmt.Errorf("cannot switch mode in a non-empty session")
	}
	s.mode = norm
	return s.LoadContext()
}

func (s *Session) IsStreaming() bool {
	return s.isStreaming
}

func (s *Session) SetStreaming(v bool) {
	s.isStreaming = v
}
