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
	CommandResultFzfMode
)

// CommandOutput is the structured result of a command execution.
type CommandOutput struct {
	Type    CommandResultType
	Payload string
}

type Session interface {
	GetMessages() []core.Message
	GetConfig() *config.Config
	SetTitle(title string)
	SetMode(mode config.AppMode) error
}

type commandFunc func(args string, s Session) (CommandOutput, bool)

type argumentCompleter func(cfg *config.Config) []string
