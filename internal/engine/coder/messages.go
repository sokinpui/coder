package coder

import (
	"fmt"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/engine/pdf"
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

	expanded := types.ExpandGroupIndices(s.messages, indices)
	if len(expanded) == 0 {
		return
	}

	repoRoot := project.Root()
	toDelete := make(map[int]struct{}, len(expanded))
	for _, idx := range expanded {
		if idx < 0 || idx >= len(s.messages) {
			continue
		}
		toDelete[idx] = struct{}{}

		msg := s.messages[idx]
		if msg.Type == types.ImageMessage && repoRoot != "" {
			imagePath := filepath.Join(repoRoot, msg.Content)
			if !strings.HasPrefix(imagePath, filepath.Join(repoRoot, ".coder", "images")) {
				continue
			}
			_ = os.Remove(imagePath)
		}
		if msg.Type == types.PDFMessage && repoRoot != "" {
			if meta, ok := types.ParsePDFMetadata(msg.Content); ok && meta.CacheDir != "" {
				pdf.EvictCache(meta.CacheDir)
				_ = os.RemoveAll(filepath.Join(repoRoot, meta.CacheDir))
			}
		}
	}

	newMessages := make([]types.Message, 0, len(s.messages)-len(toDelete))
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
