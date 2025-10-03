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

// GetPromptForTokenCount builds and returns the full prompt string for token counting.
func (s *Session) GetPromptForTokenCount() string {
	return s.modeStrategy.BuildPrompt(s.systemInstructions, s.relatedDocuments, s.projectSourceCode, s.messages)
}

// GetInitialPromptForTokenCount returns the prompt with only the context.
func (s *Session) GetInitialPromptForTokenCount() string {
	return s.modeStrategy.BuildPrompt(s.systemInstructions, s.relatedDocuments, s.projectSourceCode, nil)
}

// StartGeneration delegates to the current mode strategy to start a generation.
func (s *Session) StartGeneration() core.Event {
	return s.modeStrategy.StartGeneration(s)
}

// ProcessAIResponse delegates to the current mode strategy to process the AI response.
func (s *Session) ProcessAIResponse() core.Event {
	return s.modeStrategy.ProcessAIResponse(s)
}
