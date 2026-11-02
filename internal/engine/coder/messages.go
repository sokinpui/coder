package coder

import (
	"fmt"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
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

	repoRoot := project.Root()

	toDelete := make(map[int]struct{})
	for _, idx := range indices {
		if idx < 0 || idx >= len(s.messages) {
			continue
		}
		toDelete[idx] = struct{}{}
		for next := idx + 1; next < len(s.messages) && s.messages[next].IsDocumentImage(); next++ {
			toDelete[next] = struct{}{}
		}

		msg := s.messages[idx]
		if msg.Type == types.ImageMessage && repoRoot != "" {
			imagePath := filepath.Join(repoRoot, msg.Content)
			if !strings.HasPrefix(imagePath, filepath.Join(repoRoot, ".coder", "images")) {
				continue
			}
			_ = os.Remove(imagePath)
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
