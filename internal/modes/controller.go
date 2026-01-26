package modes

import (
	"github.com/sokinpui/coder/internal/generation"
	"github.com/sokinpui/coder/internal/types"
	"context"
)

// SessionController defines the parts of a session that a mode strategy can control.
// It is implemented by session.Session.
type SessionController interface {
	GetMessages() []types.Message
	AddMessages(msg ...types.Message)
	StartGeneration() types.Event

	// Methods needed for StartGeneration logic in strategies
	GetGenerator() *generation.Generator
	SetCancelGeneration(cancel context.CancelFunc)
	GetPrompt() string
	LoadContext() error
}
