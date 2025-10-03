package session

import (
	"coder/internal/core"
	"coder/internal/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GetMessages returns the current conversation messages.
func (s *Session) GetMessages() []core.Message {
	return s.messages
}

// AddMessage allows adding a message to the history from outside (e.g., UI-specific messages).
func (s *Session) AddMessage(msg core.Message) {
	s.messages = append(s.messages, msg)
}

// ReplaceLastMessage allows updating the last message (e.g., for streaming).
func (s *Session) ReplaceLastMessage(msg core.Message) {
	if len(s.messages) > 0 {
		s.messages[len(s.messages)-1] = msg
	}
}

// DeleteMessages removes messages at the given indices from the session.
func (s *Session) DeleteMessages(indices []int) {
	if len(indices) == 0 {
		return
	}

	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		log.Printf("Error finding repo root for deleting image: %v", err)
		// We can still proceed to delete the message from history, but log the error.
		repoRoot = ""
	}

	toDelete := make(map[int]struct{})
	for _, idx := range indices {
		if idx < 0 || idx >= len(s.messages) {
			continue
		}
		toDelete[idx] = struct{}{}

		msg := s.messages[idx]
		if msg.Type == core.ImageMessage && repoRoot != "" {
			imagePath := filepath.Join(repoRoot, msg.Content)
			// Security check to prevent path traversal
			if !strings.HasPrefix(imagePath, filepath.Join(repoRoot, ".coder", "images")) {
				log.Printf("Skipping deletion of potential path traversal: %s", msg.Content)
				continue
			}

			err := os.Remove(imagePath)
			if err != nil && !os.IsNotExist(err) {
				log.Printf("Failed to delete image file %s: %v", imagePath, err)
			}
		}
	}

	newMessages := make([]core.Message, 0, len(s.messages)-len(indices))
	for i, msg := range s.messages {
		if _, found := toDelete[i]; !found {
			newMessages = append(newMessages, msg)
		}
	}
	s.messages = newMessages
}

// EditMessage updates the content of a user message at a given index.
// It only allows editing of UserMessage types.
func (s *Session) EditMessage(index int, newContent string) error {
	if index < 0 || index >= len(s.messages) {
		return fmt.Errorf("index out of bounds: %d", index)
	}
	if s.messages[index].Type != core.UserMessage {
		return fmt.Errorf("can only edit user messages, but got type %v at index %d", s.messages[index].Type, index)
	}

	s.messages[index].Content = newContent
	return nil
}

// RemoveLastInteraction removes the last user message and AI response,
// typically after a failed or cancelled generation.
func (s *Session) RemoveLastInteraction() {
	if len(s.messages) >= 2 {
		s.messages = s.messages[:len(s.messages)-2]
	}
}
