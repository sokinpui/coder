package session

import (
	"fmt"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/types"
	"log"
	"os"
	"strings"
)

func (s *Session) hasConservation() bool {
	for _, msg := range s.messages {
		if msg.Type == types.UserMessage || msg.Type == types.ImageMessage {
			return true
		}
		if msg.Type == types.AIMessage && strings.TrimSpace(msg.Content) != "" {
			return true
		}
	}
	return false
}

func (s *Session) SaveConversation() error {
	if !s.hasConservation() && s.title == "New Chat" {
		return nil
	}

	if s.historyFilename == "" {
		s.historyFilename = fmt.Sprintf("%d.md", s.createdAt.Unix())
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Printf("could not get working directory when saving session: %v", err)
		wd = ""
	}

	data := &history.ConversationData{
		Filename:   s.historyFilename,
		Title:      s.title,
		CreatedAt:  s.createdAt,
		Messages:   s.messages,
		Context:    s.context,
		Files:      s.config.Context.Files,
		Dirs:       s.config.Context.Dirs,
		Exclusions: s.config.Context.Exclusions,
		WorkingDir: wd,
	}
	return s.historyManager.SaveConversation(data)
}

func (s *Session) GetHistoryFilename() string {
	return s.historyFilename
}

func (s *Session) LoadConversation(filename string) error {
	if len(s.messages) > 0 {
		if err := s.SaveConversation(); err != nil {
			log.Printf("Error saving current conversation before loading another: %v", err)
		}
	}

	metadata, messages, err := s.historyManager.LoadConversation(filename)
	if err != nil {
		return fmt.Errorf("failed to load conversation %s: %w", filename, err)
	}

	if metadata.WorkingDir != "" {
		if err := os.Chdir(metadata.WorkingDir); err != nil {
			log.Printf("could not switch to working directory '%s' from history file '%s': %v", metadata.WorkingDir, filename, err)
		}
	}

	s.messages = messages
	s.title = metadata.Title
	s.titleGenerated = true // A loaded conversation always has a title.
	s.createdAt = metadata.CreatedAt
	s.historyFilename = filename

	if metadata.Files != nil {
		s.config.Context.Files = metadata.Files
	} else {
		s.config.Context.Files = []string{}
	}
	if metadata.Dirs != nil {
		s.config.Context.Dirs = metadata.Dirs
	} else {
		s.config.Context.Dirs = []string{}
	}
	if metadata.Exclusions != nil {
		s.config.Context.Exclusions = metadata.Exclusions
	} else {
		s.config.Context.Exclusions = []string{}
	}

	return s.LoadContext()
}

func (s *Session) Branch(endMessageIndex int) (*Session, error) {
	if err := s.SaveConversation(); err != nil {
		return nil, fmt.Errorf("failed to save current session before branching: %w", err)
	}

	if endMessageIndex < 0 || endMessageIndex >= len(s.messages) {
		return nil, fmt.Errorf("invalid index for branching: %d", endMessageIndex)
	}

	messagesToKeep := s.messages[:endMessageIndex+1]

	newSess, err := NewWithMessages(s.config, messagesToKeep, s.modeStrategy, s.customInstruction, s.initialContextFiles)
	if err != nil {
		return nil, err
	}
	newSess.title = fmt.Sprintf("branch of %s", s.title)
	newSess.titleGenerated = true

	if err := newSess.LoadContext(); err != nil {
		return nil, fmt.Errorf("failed to load context for branched session: %w", err)
	}

	return newSess, nil
}

func (s *Session) RegenerateFrom(messageIndex int) types.Event {
	if messageIndex < 0 || messageIndex >= len(s.messages) {
		s.messages = append(s.messages, types.Message{
			Type:    types.CommandErrorResultMessage,
			Content: "Invalid index for regeneration.",
		})
		return types.Event{Type: types.MessagesUpdated}
	}
	msgType := s.messages[messageIndex].Type
	if msgType != types.UserMessage && msgType != types.ImageMessage {
		s.messages = append(s.messages, types.Message{
			Type:    types.CommandErrorResultMessage,
			Content: "Invalid index for regeneration.",
		})
		return types.Event{Type: types.MessagesUpdated}
	}

	s.messages = s.messages[:messageIndex+1]
	return s.StartGeneration()
}
