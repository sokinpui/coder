package engine

import (
	"context"
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/history"
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
	IsAgentFileRead bool
}

type EngineSession interface {
	GetID() string
	GetMode() string
	GetTitle() string
	SetTitle(title string)
	IsTitleGenerated() bool
	GenerateTitle(ctx context.Context, prompt string) string
	GetCreatedAt() time.Time
	GetHistoryFilename() string
	Capabilities() Capability

	GetMessages() []types.Message
	AddMessages(msg ...types.Message)
	PrependMessages(msg ...types.Message)
	ReplaceLastMessage(msg types.Message)
	GetPrompt() []types.Message
	DeleteMessages(indices []int)
	EditMessage(index int, newContent string) error
	SetLastModifiedFiles(files []string)
	TokenCount() int
	SetModel(model string)
	LoadContext() error
	GetContextFiles() []string

	Submit(ctx context.Context, input string) (<-chan types.SessionEvent, error)
	Cancel()
	IsStreaming() bool

	GetSupportedCommands() []string
	GetCommandDescriptions() map[string]string
	GetCommandSuggestions(cmdName, prefix string) []string
	RespondToolConfirmation(callID string, response types.ToolConfirmResponse) error
	ExecuteCommand(input string) (CommandOutput, bool)

	CreateNew(mode string) (EngineSession, error)
	SaveConversation() error
	LoadConversation(filename string) error
	Branch(endMessageIndex int) (EngineSession, error)
	Regenerate(messageIndex int) (<-chan types.SessionEvent, error)
	GetHistoryManager() *history.Manager
	GetConfig() *config.Config
	ReloadConfig() error
}
