package modes

import (
	"coder/internal/types"
	"coder/internal/generation"
	"context"
)

// SessionController defines the parts of a session that a mode strategy can control.
// It is implemented by session.Session.
type SessionController interface {
	GetMessages() []types.Message
	AddMessage(msg types.Message)
	StartGeneration() types.Event

	// Methods needed for StartGeneration logic in strategies
	GetGenerator() *generation.Generator
	SetCancelGeneration(cancel context.CancelFunc)
	GetPrompt() string
	LoadContext() error
}
