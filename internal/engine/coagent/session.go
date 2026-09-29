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
}

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
	title, err := s.Runtime.Generator.GenerateTitle(ctx, userPrompt)
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
