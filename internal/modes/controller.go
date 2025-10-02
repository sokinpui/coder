package modes

import (
	"coder/internal/core"
	"coder/internal/generation"
	"context"
)

// SessionController defines the parts of a session that a mode strategy can control.
// It is implemented by session.Session.
type SessionController interface {
	GetMessages() []core.Message
	AddMessage(msg core.Message)
	StartGeneration() core.Event

	// Methods needed for StartGeneration logic in strategies
	GetGenerator() *generation.Generator
	SetCancelGeneration(cancel context.CancelFunc)
	GetPromptForTokenCount() string
	LoadContext() error
}
