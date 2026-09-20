package engine

import (
	"context"
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/types"
)

type Session = coder.Session

type Engine interface {
	GetConfig() *config.Config
	ReloadConfig() error
	GetMode() string
	SetMode(mode string) error
	GetTitle() string
	SetTitle(title string)
	IsTitleGenerated() bool
	GenerateTitle(ctx context.Context, userPrompt string) string
	GetCreatedAt() time.Time
	GetHistoryFilename() string
	SetModel(model string)

	GetContextFiles() []string
	SetContextFiles(files []string)
	GetContextDocuments() []string
	SetContextDocuments(docs []string)
	GetDocumentPageCount(doc string) int
	GetDocumentMessages() []types.Message
	PurgeDocumentMessages(docPaths []string)
	ClearAllDocumentMessages()
	LoadContext() error
	GetLastModifiedFiles() []string
	SetLastModifiedFiles(files []string)
	HasAppliedChanges() bool
	SetHasAppliedChanges(applied bool)

	GetMessages() []types.Message
	AddMessages(msg ...types.Message)
	PrependMessages(msg ...types.Message)
	ReplaceLastMessage(msg types.Message)
	DeleteMessages(indices []int)
	EditMessage(index int, newContent string) error
	HasChatHistory() bool

	HandleInput(input string) types.Event
	StartGeneration() types.Event
	CancelGeneration()
	IsStreaming() bool
	SetStreaming(v bool)
	RegenerateFrom(messageIndex int) types.Event
	SaveConversation() error
	LoadConversation(filename string) error
	GetPrompt() []types.Message
	GetHistoryManager() *history.Manager
}

func New(cfg *config.Config, mode string, instruction string, contextFiles []string) (*coder.Session, error) {
	return coder.New(cfg, mode, instruction, contextFiles)
}

func NewWithMessages(cfg *config.Config, initialMessages []types.Message, mode string, instruction string, contextFiles []string) (*coder.Session, error) {
	return coder.NewWithMessages(cfg, initialMessages, mode, instruction, contextFiles)
}
