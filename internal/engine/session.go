package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/types"
)

const (
	ModeCoder = "coder"
	ModeChat  = "chat"
	ModeAgent = "agent"
)

type SessionFactory func(cfg *config.Config, mode string, instruction string, contextFiles []string) (EngineSession, error)

var sessionFactory SessionFactory

func RegisterSessionFactory(fn SessionFactory) {
	sessionFactory = fn
}

func NewSession(cfg *config.Config, mode string, instruction string, contextFiles []string) (EngineSession, error) {
	if sessionFactory == nil {
		return nil, fmt.Errorf("session factory not registered")
	}
	return sessionFactory(cfg, NormalizeMode(mode), instruction, contextFiles)
}

func NormalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ModeChat:
		return ModeChat
	case ModeAgent:
		return ModeAgent
	default:
		return ModeCoder
	}
}

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

type ContextController interface {
	LoadContext() error
	GetContextFiles() []string
	SetContextFiles(files []string)
	GetContextDocuments() []string
	SetContextDocuments(docs []string)
	GetDocumentPageCount(doc string) int
	PurgeDocumentMessages(docPaths []string)
	ClearAllDocumentMessages()
}

type ChangeApplier interface {
	GetLastModifiedFiles() []string
	SetLastModifiedFiles(files []string)
	HasAppliedChanges() bool
	SetHasAppliedChanges(applied bool)
}

type ToolApprover interface {
	RespondToolConfirmation(callID string, response types.ToolConfirmResponse) error
}

type AgentFileReader interface {
	ReadAgentFiles(input string, paths []string) (CommandOutput, bool)
}

type EngineSession interface {
	GetID() string
	GetMode() string
	GetTitle() string
	SetTitle(title string)
	IsTitleGenerated() bool
	GetInstruction() string
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
	TokenCount() int
	GetReasoningEffort() string
	SetReasoningEffort(effort string)
	SetModel(model string)

	Submit(ctx context.Context, input string) (<-chan types.SessionEvent, error)
	Cancel()
	IsStreaming() bool

	GetSupportedCommands() []string
	GetCommandDescriptions() map[string]string
	GetCommandSuggestions(cmdName, prefix string) []string
	ExecuteCommand(input string) (CommandOutput, bool)

	CreateNew(mode string) (EngineSession, error)
	SaveConversation() error
	LoadConversation(filename string) error
	Branch(endMessageIndex int) (EngineSession, error)
	Regenerate(messageIndex int) (<-chan types.SessionEvent, error)
	GetHistoryManager() *history.Manager
	GetConfig() *config.Config
	ReloadConfig() error
	HasChatHistory() bool
}
