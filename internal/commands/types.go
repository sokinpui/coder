package commands

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

type CommandResultType int

const (
	CommandResultString CommandResultType = iota
	CommandResultNewSession
	CommandResultChatMode
	CommandResultGenerateMode
	CommandResultVisualMode
	CommandResultEditMode
	CommandResultBranchMode
	CommandResultSearchMode
	CommandResultHistoryMode
	CommandResultFzfMode
	CommandResultTreeMode
	CommandResultQuit
)

// CommandOutput is the structured result of a command execution.
type CommandOutput struct {
	Type    CommandResultType
	Payload string
}

type SessionController interface {
	GetMessages() []types.Message
	GetConfig() *config.Config
	SetTitle(title string)
	LoadContext() error
}

type commandFunc func(args string, s SessionController) (CommandOutput, bool)

type argumentCompleter func(cfg *config.Config) []string
