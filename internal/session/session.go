package session

import (
	"context"
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/generation"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ModeCoding = "coding"
	ModeChat   = "chat"
)

type Session struct {
	ID                string
	config            *config.Config
	generator         *generation.Generator
	historyManager    *history.Manager
	messages          []types.Message
	cancelGeneration  context.CancelFunc
	title             string
	titleGenerated    bool
	historyFilename   string
	createdAt         time.Time
	mode              string
	instruction       string
	projectSourceCode string
	lastModifiedFiles []string
	hasAppliedChanges bool
	contextFiles      []string
	contextDocuments  []string
	documentMessages  []types.Message
	cachedDocMessages map[string][]types.Message
	docModTimes       map[string]time.Time
	contextLoadedAt   time.Time
	isStreaming       bool
}

func New(cfg *config.Config, mode string, instruction string, contextFiles []string) (*Session, error) {
	if mode == "" {
		mode = ModeCoding
	}
	return NewWithMessages(cfg, nil, mode, instruction, contextFiles)
}

func NewWithMessages(cfg *config.Config, initialMessages []types.Message, mode string, instruction string, contextFiles []string) (*Session, error) {
	if mode == "" {
		mode = ModeCoding
	}
	hist, err := history.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history manager: %w", err)
	}

	messages := make([]types.Message, len(initialMessages))
	copy(messages, initialMessages)

	cfgCopy := *cfg
	cfgCopy.Context.Files = append([]string{}, cfg.Context.Files...)
	cfgCopy.Context.Dirs = append([]string{}, cfg.Context.Dirs...)
	cfgCopy.Context.Exclusions = append([]string{}, cfg.Context.Exclusions...)

	gen, err := generation.New(&cfgCopy)
	if err != nil {
		return nil, err
	}

	allExclusions := append([]string{}, source.Exclusions...)
	allExclusions = append(allExclusions, cfgCopy.Context.Exclusions...)

	var cleanContextFiles []string
	var pdfFiles []string
	for _, p := range contextFiles {
		if strings.EqualFold(filepath.Ext(p), ".pdf") {
			pdfFiles = append(pdfFiles, filepath.ToSlash(p))
			continue
		}
		cleanContextFiles = append(cleanContextFiles, p)
	}

	var resolvedContextFiles []string
	switch mode {
	case ModeCoding:
		var dirs, files []string

		if len(cleanContextFiles) > 0 {
			for _, p := range cleanContextFiles {
				p = filepath.ToSlash(p)
				if info, err := os.Stat(p); err == nil && info.IsDir() {
					dirs = append(dirs, p)
				} else {
					files = append(files, p)
				}
			}
		} else {
			dirs = getSafeContextDirs(cfgCopy.Context.Dirs)
			files = cfgCopy.Context.Files
		}

		resolvedContextFiles, _ = utils.SourceToFileList(dirs, files, allExclusions)
	default:
		// Other modes do not load context files by default
	}

	s := &Session{
		ID:                fmt.Sprintf("%d", time.Now().UnixNano()),
		config:            &cfgCopy,
		generator:         gen,
		historyManager:    hist,
		messages:          messages,
		title:             "New Chat",
		titleGenerated:    false,
		createdAt:         time.Now(),
		historyFilename:   "",
		mode:              mode,
		instruction:       instruction,
		contextFiles:      resolvedContextFiles,
		contextDocuments:  AppendUnique([]string{}, pdfFiles),
		cachedDocMessages: make(map[string][]types.Message),
		docModTimes:       make(map[string]time.Time),
	}

	return s, nil
}

func getSafeContextDirs(configuredDirs []string) []string {
	if utils.IsGitRepo() {
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
	s.generator.Config = cfg.Generation
	s.generator.BaseURL = cfg.Server.URL
	s.generator.APIKey = cfg.Server.APIKey
	return nil
}

func (s *Session) GetGenerator() *generation.Generator {
	return s.generator
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
	seen := make(map[string]struct{}, len(original)+len(newItems))
	for _, item := range original {
		seen[item] = struct{}{}
	}
	result := append([]string{}, original...)
	for _, item := range newItems {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
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
		return ModeCoding
	}
	return s.mode
}

func (s *Session) SetModel(model string) {
	s.config.Generation.ModelCode = model
	s.generator.Config.ModelCode = model
}

func (s *Session) SetMode(mode string) error {
	s.mode = mode
	return s.LoadContext()
}

func (s *Session) IsStreaming() bool {
	return s.isStreaming
}

func (s *Session) SetStreaming(v bool) {
	s.isStreaming = v
}
