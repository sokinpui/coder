package session

import (
	"context"
	"github.com/sokinpui/coder/internal/types"
)

func (s *Session) SetCancelGeneration(cancel context.CancelFunc) {
	s.cancelGeneration = cancel
}

func (s *Session) CancelGeneration() {
	if s.cancelGeneration != nil {
		s.cancelGeneration()
	}
}

func (s *Session) GetPrompt() []types.Message {
	return s.modeStrategy.BuildPrompt(s.messages)
}

func (s *Session) StartGeneration() types.Event {
	return s.modeStrategy.StartGeneration(s)
}
