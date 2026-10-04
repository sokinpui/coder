package commands

import (
	"github.com/sokinpui/coder/internal/engine"
)

type CommandOutput = engine.CommandOutput

type SessionController interface {
	engine.EngineSession
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
	HasChatHistory() bool
	ReadAgentFiles(input string, paths []string) (CommandOutput, bool)
}

type commandFunc func(args string, s SessionController) (CommandOutput, bool)

type argumentCompleter func(s SessionController, prefix string) []string
