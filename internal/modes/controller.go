package modes

import (
	"context"
	"github.com/sokinpui/coder/internal/generation"
	"github.com/sokinpui/coder/internal/types"
)

type SessionController interface {
	GetMessages() []types.Message
	AddMessages(msg ...types.Message)
	StartGeneration() types.Event

	GetGenerator() *generation.Generator
	SetCancelGeneration(cancel context.CancelFunc)
	GetPrompt() string
	LoadContext() error
}
