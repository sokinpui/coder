package session

import (
	"context"
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/generation"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/modes"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"os"
	"time"
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
	modeStrategy      modes.ModeStrategy
	instruction       string
	lastModifiedFiles []string
	hasAppliedChanges bool
	contextFiles      []string
}

func New(cfg *config.Config, mode string, instruction string, contextFiles []string) (*Session, error) {
	strategy := modes.GetStrategy(mode, instruction)
	return NewWithMessages(cfg, nil, strategy, instruction, contextFiles)
}

func NewWithMessages(cfg *config.Config, initialMessages []types.Message, strategy modes.ModeStrategy, instruction string, contextFiles []string) (*Session, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return nil, err
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

	allExclusions := append([]string{}, source.Exclusions...)
	allExclusions = append(allExclusions, cfgCopy.Context.Exclusions...)

	var resolvedContextFiles []string
	// Only resolve context if we are NOT in chat mode
	if _, isChat := strategy.(*modes.ChatMode); !isChat {
		if len(contextFiles) > 0 {
			var dirs, files []string
			for _, p := range contextFiles {
				if info, err := os.Stat(p); err == nil && info.IsDir() {
					dirs = append(dirs, p)
					continue
				}
				files = append(files, p)
			}
			resolvedContextFiles, _ = utils.SourceToFileList(dirs, files, allExclusions)
		} else {
			resolvedContextFiles, _ = utils.SourceToFileList(cfgCopy.Context.Dirs, cfgCopy.Context.Files, allExclusions)
		}
	}

	s := &Session{
		ID:              fmt.Sprintf("%d", time.Now().UnixNano()),
		config:          &cfgCopy,
		generator:       gen,
		historyManager:  hist,
		messages:        messages,
		title:           "New Chat",
		titleGenerated:  false,
		createdAt:       time.Now(),
		historyFilename: "",
		modeStrategy:    strategy,
		instruction:     instruction,
		contextFiles:    resolvedContextFiles,
	}

	return s, nil
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

func (s *Session) GetInstruction() string {
	return s.instruction
}

func (s *Session) GetContextFiles() []string {
	return s.contextFiles
}

func (s *Session) SetContextFiles(files []string) {
	s.contextFiles = files
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
