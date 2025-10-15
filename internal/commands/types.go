package commands

import (
	"coder/internal/config"
	"coder/internal/core"
)

type CommandResultType int

const (
	CommandResultString CommandResultType = iota
	CommandResultNewSession
	CommandResultGenerateMode
	CommandResultVisualMode
	CommandResultEditMode
	CommandResultBranchMode
	CommandResultHistoryMode
)

// CommandOutput is the structured result of a command execution.
type CommandOutput struct {
	Type    CommandResultType
	Payload string
}

type SessionController interface {
	GetMessages() []core.Message
	GetConfig() *config.Config
	SetTitle(title string)
	SetMode(mode config.AppMode) error
	LoadContext() error
}

type commandFunc func(args string, s SessionController) (CommandOutput, bool)

type argumentCompleter func(cfg *config.Config) []string
