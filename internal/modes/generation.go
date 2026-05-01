package modes

import (
	"context"
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"log"
	"os"
	"path/filepath"
)

func StartGeneration(s SessionController, generationConfig *config.Generation) types.Event {
	// Reload context, which includes project source, before every generation
	// to pick up any file changes.
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

	// Populate image data
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

	streamChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	s.SetCancelGeneration(cancel)
	go s.GetGenerator().GenerateTask(ctx, messages, streamChan, generationConfig)

	s.AddMessages(types.Message{Type: types.AIMessage, Content: ""}) // Placeholder for AI

	return types.Event{
		Type: types.GenerationStarted,
		Data: streamChan,
	}
}
