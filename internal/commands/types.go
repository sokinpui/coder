package commands

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

// CommandOutput is the structured result of a command execution.
type CommandOutput struct {
	Type    types.EventType
	Payload string
}

type SessionController interface {
	GetMessages() []types.Message
	GetConfig() *config.Config
	SetTitle(title string)
	LoadContext() error
	NewSession()
	StartChatSession()
}

type commandFunc func(args string, s SessionController) (CommandOutput, bool)

type argumentCompleter func(cfg *config.Config) []string
