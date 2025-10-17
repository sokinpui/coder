package session

import (
	"coder/internal/config"
	"coder/internal/history"
	"coder/internal/types"
	"fmt"
	"log"
	"os"
	"time"
)

// SaveConversation saves the current conversation to history.
func (s *Session) SaveConversation() error {
	historyContent := history.BuildHistorySnippet(s.messages)

	// Don't save a session if it's a fresh, unmodified one.
	if historyContent == "" && s.title == "New Chat" {
		return nil
	}

	if s.historyFilename == "" {
		s.historyFilename = fmt.Sprintf("%d.md", s.createdAt.Unix())
	}

	wd, err := os.Getwd()
	if err != nil {
		// Log the error but don't fail the save.
		log.Printf("could not get working directory when saving session: %v", err)
		wd = ""
	}

	data := &history.ConversationData{
		Filename:   s.historyFilename,
		Title:      s.title,
		CreatedAt:  s.createdAt,
		Messages:   s.messages,
		Context:    s.context,
		Files:      s.config.Sources.Files,
		Dirs:       s.config.Sources.Dirs,
		WorkingDir: wd,
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

	// Change directory if specified in history.
	if metadata.WorkingDir != "" {
		if err := os.Chdir(metadata.WorkingDir); err != nil {
			// Log the error but continue. The user might have moved the project.
			log.Printf("could not switch to working directory '%s' from history file '%s': %v", metadata.WorkingDir, filename, err)
		}
	}

	s.messages = messages
	s.title = metadata.Title
	s.titleGenerated = true // A loaded conversation always has a title.
	s.createdAt = metadata.CreatedAt
	s.historyFilename = filename

	// Update sources from history. If not present in history (e.g. old format),
	// clear them to match the state when the conversation was saved.
	if metadata.Files != nil {
		s.config.Sources.Files = metadata.Files
	} else {
		s.config.Sources.Files = []string{}
	}
	if metadata.Dirs != nil {
		s.config.Sources.Dirs = metadata.Dirs
	} else {
		s.config.Sources.Dirs = []string{}
	}

	// The context, including project source, is loaded based on the current mode.
	return s.LoadContext()
}

func (s *Session) newSession() {
	if err := s.SaveConversation(); err != nil {
		log.Printf("Error saving conversation for /new command: %v", err)
	}
	s.messages = []types.Message{} // Clear messages
	s.title = "New Chat"
	s.titleGenerated = false
	s.createdAt = time.Now()
	s.historyFilename = ""
	defaultConfig := config.Default()
	s.config.Sources = defaultConfig.Sources
	if err := s.LoadContext(); err != nil {
		// Log and add an error message for the user to see.
		log.Printf("Error reloading context for new session: %v", err)
		s.messages = append(s.messages, types.Message{Type: types.CommandErrorResultMessage, Content: fmt.Sprintf("Failed to reload context for new session: %v", err)})
	}
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
