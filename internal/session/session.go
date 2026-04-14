package session

import (
	"context"
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/generation"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/modes"
	"github.com/sokinpui/coder/internal/types"
	"time"
)

type Session struct {
	ID                  string
	config              *config.Config
	generator           *generation.Generator
	historyManager      *history.Manager
	messages            []types.Message
	cancelGeneration    context.CancelFunc
	title               string
	context             string // Role instruction + source code
	titleGenerated      bool
	historyFilename     string
	createdAt           time.Time
	modeStrategy        modes.ModeStrategy
	customInstruction   string
	initialContextFiles []string
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

	s := &Session{
		ID:                  fmt.Sprintf("%d", time.Now().UnixNano()),
		config:              &cfgCopy,
		generator:           gen,
		historyManager:      hist,
		messages:            messages,
		title:               "New Chat",
		titleGenerated:      false,
		createdAt:           time.Now(),
		historyFilename:     "",
		modeStrategy:        strategy,
		customInstruction:   instruction,
		initialContextFiles: contextFiles,
	}

	if len(contextFiles) > 0 {
		s.applyContextFiles(contextFiles)
	}
	return s, nil
}

func (s *Session) GetConfig() *config.Config {
	return s.config
}

func (s *Session) GetGenerator() *generation.Generator {
	return s.generator
}

func (s *Session) GetHistoryManager() *history.Manager {
	return s.historyManager
}

func (s *Session) GetCustomInstruction() string {
	return s.customInstruction
}

func (s *Session) GetInitialContextFiles() []string {
	return s.initialContextFiles
}
