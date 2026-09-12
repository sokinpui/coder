package commands

import (
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

type CommandOutput struct {
	Type            types.EventType
	Payload         string
	Mode            string
	IsShell         bool
	IsContext       bool
	IsFileApply     bool
	IsFileApplyUndo bool
}

type SessionController interface {
	GetMessages() []types.Message
	AddMessages(msg ...types.Message)
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
	GetContextDocuments() []string
	SetContextDocuments(docs []string)
	GetDocumentPageCount(doc string) int
	PurgeDocumentMessages(docPaths []string)
	ClearAllDocumentMessages()
	GetMode() string
	SetMode(mode string) error
	SetModel(model string)
	IsToolsEnabled() bool
	SetToolsEnabled(enabled bool)
	HasChatHistory() bool
}

type commandFunc func(args string, s SessionController) (CommandOutput, bool)

type argumentCompleter func(cfg *config.Config, prefix string) []string
