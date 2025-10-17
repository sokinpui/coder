package session

import (
	"coder/internal/types"
	"coder/internal/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (s *Session) GetMessages() []types.Message {
	return s.messages
}

func (s *Session) AddMessages(msg ...types.Message) {
	s.messages = append(s.messages, msg...)
}

func (s *Session) PrependMessages(msg ...types.Message) {
	s.messages = append(msg, s.messages...)
}

func (s *Session) ReplaceLastMessage(msg types.Message) {
	if len(s.messages) > 0 {
		s.messages[len(s.messages)-1] = msg
	}
}

func (s *Session) DeleteMessages(indices []int) {
	if len(indices) == 0 {
		return
	}

	repoRoot, err := utils.FindRepoRoot()
	if err != nil {
		log.Printf("Error finding repo root for deleting image: %v", err)
		repoRoot = ""
	}

	toDelete := make(map[int]struct{})
	for _, idx := range indices {
		if idx < 0 || idx >= len(s.messages) {
			continue
		}
		toDelete[idx] = struct{}{}

		msg := s.messages[idx]
		if msg.Type == types.ImageMessage && repoRoot != "" {
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

	newMessages := make([]types.Message, 0, len(s.messages)-len(indices))
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
	if s.messages[index].Type != types.UserMessage {
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
