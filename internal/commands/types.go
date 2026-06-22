package commands

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

type CommandOutput struct {
	Type     types.EventType
	Payload  string
	Mode     string
	Metadata map[string]any
}

type SessionController interface {
	GetMessages() []types.Message
	GetConfig() *config.Config
	SetTitle(title string)
	ReloadConfig() error
	LoadContext() error
	GetLastModifiedFiles() []string
	SetLastModifiedFiles(files []string)
	HasAppliedChanges() bool
	SetHasAppliedChanges(applied bool)
	GetContextFiles() []string
	SetContextFiles(files []string)
}

type commandFunc func(args string, s SessionController) (CommandOutput, bool)

type argumentCompleter func(cfg *config.Config, prefix string) []string
