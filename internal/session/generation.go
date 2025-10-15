package session

import (
	"coder/internal/core"
	"context"
)

// SetCancelGeneration sets the context cancel function for the current generation.
func (s *Session) SetCancelGeneration(cancel context.CancelFunc) {
	s.cancelGeneration = cancel
}

// CancelGeneration cancels any ongoing AI generation.
func (s *Session) CancelGeneration() {
	if s.cancelGeneration != nil {
		s.cancelGeneration()
	}
}

// GetPrompt builds and returns the full prompt string for token counting.
func (s *Session) GetPrompt() string {
	return s.modeStrategy.BuildPrompt(s.messages)
}

// StartGeneration delegates to the current mode strategy to start a generation.
func (s *Session) StartGeneration() core.Event {
	return s.modeStrategy.StartGeneration(s)
}
