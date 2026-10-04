package coagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/config"
	coagentprompt "github.com/sokinpui/coder/internal/engine/coagent/prompt"
	coderprompt "github.com/sokinpui/coder/internal/engine/coder/prompt"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

const ModeCoAgent = "coagent"

type Session struct {
	ID              string
	Config          *config.Config
	Runtime         *AgentRuntime
	HistoryManager  *history.Manager
	Messages        []types.Message
	Title           string
	TitleGenerated  bool
	HistoryFilename string
	CreatedAt       time.Time
	Instruction     string
	isStreaming     bool
	cancelFunc      context.CancelFunc
}

var _ commands.SessionController = (*Session)(nil)

func NewSession(cfg *config.Config) (*Session, error) {
	hist, err := history.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to init history manager: %w", err)
	}

	rt, err := NewAgentRuntime(cfg, nil)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:             fmt.Sprintf("%d", time.Now().UnixNano()),
		Config:         cfg,
		Runtime:        rt,
		HistoryManager: hist,
		Title:          "New Agent Session",
		CreatedAt:      time.Now(),
	}, nil
}

func (s *Session) GetID() string {
	return s.ID
}

func (s *Session) GetTitle() string {
	return s.Title
}

func (s *Session) GetCreatedAt() time.Time {
	return s.CreatedAt
}

func (s *Session) GetHistoryFilename() string {
	return s.HistoryFilename
}

func (s *Session) GetConfig() *config.Config {
	return s.Config
}

func (s *Session) SetModel(model string) {
	s.Config.Agent.ModelCode = model
	s.Runtime.Config.ModelCode = model
}

func (s *Session) AddMessages(msg ...types.Message) {
	s.Messages = append(s.Messages, msg...)
}

func (s *Session) PrependMessages(msg ...types.Message) {
	s.Messages = append(msg, s.Messages...)
}

func (s *Session) ReplaceLastMessage(msg types.Message) {
	if len(s.Messages) > 0 {
		s.Messages[len(s.Messages)-1] = msg
	}
}

func (s *Session) LoadContext() error                      { return nil }
func (s *Session) GetLastModifiedFiles() []string          { return nil }
func (s *Session) SetLastModifiedFiles(files []string)     {}
func (s *Session) HasAppliedChanges() bool                 { return false }
func (s *Session) SetHasAppliedChanges(applied bool)       {}
func (s *Session) GetContextFiles() []string               { return nil }
func (s *Session) SetContextFiles(files []string)          {}
func (s *Session) GetContextDocuments() []string           { return nil }
func (s *Session) SetContextDocuments(docs []string)       {}
func (s *Session) GetDocumentPageCount(doc string) int     { return 0 }
func (s *Session) PurgeDocumentMessages(docPaths []string) {}
func (s *Session) ClearAllDocumentMessages()               {}
func (s *Session) SetMode(mode string) error               { return nil }
func (s *Session) HasChatHistory() bool {
	return len(s.Messages) > 0 || s.HistoryFilename != ""
}

func (s *Session) GetPrompt() []types.Message {
	instr := s.Instruction
	if instr == "" {
		instr = coagentprompt.Instructions
	}
	prompt := []types.Message{
		{Type: types.InstructionMessage, Content: instr},
	}
	prompt = append(prompt, s.PrepareMessages()...)
	return prompt
}

func (s *Session) PrepareMessages() []types.Message {
	repoRoot := project.Root()
	prepared := make([]types.Message, len(s.Messages))
	copy(prepared, s.Messages)

	for i := range prepared {
		if prepared[i].Type == types.ImageMessage && len(prepared[i].Data) == 0 && prepared[i].Content != "" {
			absPath := prepared[i].Content
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(repoRoot, absPath)
			}
			if data, err := os.ReadFile(absPath); err == nil {
				prepared[i].Data = data
			}
		}
	}
	return prepared
}

func (s *Session) SaveConversation() error {
	if len(s.Messages) == 0 {
		return nil
	}

	if s.HistoryFilename == "" {
		s.HistoryFilename = fmt.Sprintf("%d.md", s.CreatedAt.Unix())
	}

	wd, _ := os.Getwd()
	data := &history.ConversationData{
		Filename:   s.HistoryFilename,
		Title:      s.Title,
		Mode:       ModeCoAgent,
		CreatedAt:  s.CreatedAt,
		Messages:   s.Messages,
		WorkingDir: wd,
	}
	return s.HistoryManager.SaveConversation(data)
}

func (s *Session) LoadConversation(filename string) error {
	metadata, msgs, err := s.HistoryManager.LoadConversation(filename)
	if err != nil {
		return err
	}

	s.Messages = msgs
	s.Title = metadata.Title
	s.TitleGenerated = true
	s.CreatedAt = metadata.CreatedAt
	s.HistoryFilename = filename
	return nil
}

func (s *Session) GenerateTitle(ctx context.Context, userPrompt string) string {
	s.TitleGenerated = true
	prompt := strings.Replace(coderprompt.TitleGenerationPrompt, "{{PROMPT}}", userPrompt, 1)
	title, err := s.Runtime.Generator.GenerateTitle(ctx, prompt)
	if err != nil {
		words := strings.Fields(userPrompt)
		numWords := min(len(words), 5)
		fallback := strings.Join(words[:numWords], " ")
		if len(words) > numWords {
			fallback += "..."
		}
		s.Title = fallback
		return s.Title
	}

	s.Title = strings.Trim(strings.TrimSpace(title), "\"")
	return s.Title
}
