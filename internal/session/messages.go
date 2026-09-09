package session

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
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

	repoRoot := utils.GetProjectRoot()

	toDelete := make(map[int]struct{})
	for _, idx := range indices {
		if idx < 0 || idx >= len(s.messages) {
			continue
		}
		toDelete[idx] = struct{}{}

		msg := s.messages[idx]
		if msg.Type == types.ImageMessage && repoRoot != "" {
			imagePath := filepath.Join(repoRoot, msg.Content)
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

func (s *Session) PurgeDocumentMessages(docPaths []string) {
	if len(docPaths) == 0 || len(s.messages) == 0 {
		return
	}

	docNames := make(map[string]struct{}, len(docPaths))
	for _, p := range docPaths {
		base := filepath.Base(p)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		docNames[name] = struct{}{}
	}

	filtered := make([]types.Message, 0, len(s.messages))
	for _, msg := range s.messages {
		if msg.Type != types.ImageMessage {
			filtered = append(filtered, msg)
			continue
		}
		slashed := filepath.ToSlash(msg.Content)
		isDocImage := false
		for name := range docNames {
			if strings.Contains(slashed, "/"+name+"/") {
				isDocImage = true
				break
			}
		}
		if !isDocImage {
			filtered = append(filtered, msg)
		}
	}
	s.messages = filtered
}

func (s *Session) ClearAllDocumentMessages() {
	if len(s.messages) == 0 {
		return
	}
	filtered := make([]types.Message, 0, len(s.messages))
	for _, msg := range s.messages {
		if msg.Type == types.ImageMessage {
			slashed := filepath.ToSlash(msg.Content)
			parts := strings.Split(slashed, "/")
			if len(parts) >= 3 && parts[len(parts)-3] == "images" {
				continue
			}
		}
		filtered = append(filtered, msg)
	}
	s.messages = filtered
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
