package session

import (
	"coder/internal/types"
	"coder/internal/config"
	"coder/internal/generation"
	"coder/internal/history"
	"coder/internal/modes"
	"context"
	"fmt"
	"time"
)

type Session struct {
	config           *config.Config
	generator        *generation.Generator
	historyManager   *history.Manager
	messages         []types.Message
	cancelGeneration context.CancelFunc
	title            string
	context          string // Role instruction + source code
	titleGenerated   bool
	historyFilename  string
	createdAt        time.Time
	modeStrategy     modes.ModeStrategy
}

func New(cfg *config.Config) (*Session, error) {
	return NewWithMessages(cfg, nil)
}

// NewWithMessages creates a new session with a pre-existing set of messages.
func NewWithMessages(cfg *config.Config, initialMessages []types.Message) (*Session, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return nil, err
	}

	hist, err := history.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history manager: %w", err)
	}

	// Make a defensive copy of the slice to avoid external modifications.
	messages := make([]types.Message, len(initialMessages))
	copy(messages, initialMessages)

	return &Session{
		config:          cfg,
		generator:       gen,
		historyManager:  hist,
		messages:        messages,
		title:           "New Chat",
		titleGenerated:  false,
		createdAt:       time.Now(),
		historyFilename: "",
		modeStrategy:    modes.NewStrategy(cfg.AppMode),
	}, nil
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
