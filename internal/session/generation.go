package session

import (
	"context"
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"log"
	"os"
	"path/filepath"
)

func (s *Session) SetCancelGeneration(cancel context.CancelFunc) {
	s.cancelGeneration = cancel
}

func (s *Session) CancelGeneration() {
	if s.cancelGeneration != nil {
		s.cancelGeneration()
	}
}

func (s *Session) GetPrompt() []types.Message {
	return s.BuildPrompt(s.messages)
}

func (s *Session) StartGeneration() types.Event {
	if err := s.LoadContext(); err != nil {
		log.Printf("Error reloading context for generation: %v", err)
		s.AddMessages(types.Message{
			Type:    types.CommandErrorResultMessage,
			Content: fmt.Sprintf("Failed to reload context before generation:\n%v", err),
		})
		return types.Event{Type: types.MessagesUpdated}
	}

	messages := s.GetPrompt()
	repoRoot := utils.GetProjectRoot()

	for i := range messages {
		if messages[i].Type == types.ImageMessage && messages[i].Data == nil {
			absPath := filepath.Join(repoRoot, messages[i].Content)
			data, err := os.ReadFile(absPath)
			if err != nil {
				log.Printf("Error reading image file %s: %v", absPath, err)
				continue
			}
			messages[i].Data = data
		}
	}

	streamChan := make(chan types.StreamChunk, 100)
	ctx, cancel := context.WithCancel(context.Background())
	s.SetCancelGeneration(cancel)
	go s.generator.GenerateTask(ctx, messages, streamChan, nil)

	s.AddMessages(types.Message{Type: types.AIMessage, Content: ""})

	return types.Event{
		Type: types.GenerationStarted,
		Data: streamChan,
	}
}
