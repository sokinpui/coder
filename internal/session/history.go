package session

import (
	"coder/internal/core"
	"coder/internal/history"
	"fmt"
	"log"
	"time"
)

// SaveConversation saves the current conversation to history.
func (s *Session) SaveConversation() error {
	if s.historyFilename == "" {
		s.historyFilename = fmt.Sprintf("%d.md", s.createdAt.Unix())
	}

	preamble := s.modeStrategy.BuildPrompt(nil)
	data := &history.ConversationData{
		Filename:  s.historyFilename,
		Title:     s.title,
		CreatedAt: s.createdAt,
		Messages:  s.messages,
		Preamble:  preamble,
		FilePaths: s.config.Sources.FilePaths,
		FileDirs:  s.config.Sources.FileDirs,
	}
	return s.historyManager.SaveConversation(data)
}

// GetHistoryFilename returns the filename for the current conversation in history.
// It returns an empty string if the session hasn't been saved yet.
func (s *Session) GetHistoryFilename() string {
	return s.historyFilename
}

// LoadConversation loads a conversation from a history file, replacing the current session state.
func (s *Session) LoadConversation(filename string) error {
	if len(s.messages) > 0 {
		if err := s.SaveConversation(); err != nil {
			// Log the error but continue, as loading a new session is more important.
			log.Printf("Error saving current conversation before loading another: %v", err)
		}
	}

	metadata, messages, err := s.historyManager.LoadConversation(filename)
	if err != nil {
		return fmt.Errorf("failed to load conversation %s: %w", filename, err)
	}

	s.messages = messages
	s.title = metadata.Title
	s.titleGenerated = true // A loaded conversation always has a title.
	s.createdAt = metadata.CreatedAt
	s.historyFilename = filename

	// Update sources from history. If not present in history (e.g. old format),
	// clear them to match the state when the conversation was saved.
	if metadata.FilePaths != nil {
		s.config.Sources.FilePaths = metadata.FilePaths
	} else {
		s.config.Sources.FilePaths = []string{}
	}
	if metadata.FileDirs != nil {
		s.config.Sources.FileDirs = metadata.FileDirs
	} else {
		s.config.Sources.FileDirs = []string{}
	}

	// The context, including project source, is loaded based on the current mode.
	return s.LoadContext()
}

func (s *Session) newSession() {
	if err := s.SaveConversation(); err != nil {
		log.Printf("Error saving conversation for /new command: %v", err)
	}
	s.messages = []core.Message{} // Clear messages
	s.title = "New Chat"
	s.titleGenerated = false
	s.createdAt = time.Now()
	s.historyFilename = ""
}

// Branch saves the current session and creates a new one containing messages
// up to the specified index.
func (s *Session) Branch(endMessageIndex int) (*Session, error) {
	if err := s.SaveConversation(); err != nil {
		return nil, fmt.Errorf("failed to save current session before branching: %w", err)
	}

	if endMessageIndex < 0 || endMessageIndex >= len(s.messages) {
		return nil, fmt.Errorf("invalid index for branching: %d", endMessageIndex)
	}

	messagesToKeep := s.messages[:endMessageIndex+1]

	// NewWithMessages makes a defensive copy, so this is safe.
	newSess, err := NewWithMessages(s.config, messagesToKeep)
	if err != nil {
		return nil, err
	}

	if err := newSess.LoadContext(); err != nil {
		return nil, fmt.Errorf("failed to load context for branched session: %w", err)
	}

	return newSess, nil
}

// RegenerateFrom truncates the message history to the specified user message
// and starts a new generation.
func (s *Session) RegenerateFrom(userMessageIndex int) core.Event {
	if userMessageIndex < 0 || userMessageIndex >= len(s.messages) || s.messages[userMessageIndex].Type != core.UserMessage {
		s.messages = append(s.messages, core.Message{
			Type:    core.CommandErrorResultMessage,
			Content: "Invalid index for regeneration.",
		})
		return core.Event{Type: core.MessagesUpdated}
	}

	s.messages = s.messages[:userMessageIndex+1]
	return s.StartGeneration()
}
