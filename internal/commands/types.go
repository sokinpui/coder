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

// SessionChanger is an interface that allows commands to modify session state
// without creating a circular dependency between core and session packages.
type SessionChanger interface {
	SetTitle(title string)
	SetMode(mode config.AppMode) error
}

type commandFunc func(args string, messages []core.Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool)

type argumentCompleter func(cfg *config.Config) []string
