package session

import (
	"coder/internal/core"
	"coder/internal/modes"
	"strings"
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

// GetPromptForTokenCount builds and returns the full prompt string for token counting.
func (s *Session) GetPromptForTokenCount() string {
	if len(s.messages) == 0 {
		return s.preamble
	}

	historySection := modes.ConversationHistorySection(s.messages)

	var sb strings.Builder
	sb.WriteString(s.preamble)
	if s.preamble != "" {
		sb.WriteString(modes.Separator)
	}
	sb.WriteString(historySection.Header)
	sb.WriteString(historySection.Content)
	return sb.String()
}

// GetInitialPromptForTokenCount returns the prompt with only the context.
func (s *Session) GetInitialPromptForTokenCount() string {
	return s.preamble
}

// StartGeneration delegates to the current mode strategy to start a generation.
func (s *Session) StartGeneration() core.Event {
	return s.modeStrategy.StartGeneration(s)
}
