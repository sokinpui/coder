package modes

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/utils"
	"context"
	"fmt"
	"log"
	"path/filepath"
)

// StartGeneration provides a default implementation for starting a generation task,
// optionally allowing a specific generation config to be used.
func StartGeneration(s SessionController, generationConfig *config.Generation) core.Event {
	// Reload context, which includes project source, before every generation
	// to pick up any file changes.
	if err := s.LoadContext(); err != nil {
		log.Printf("Error reloading context for generation: %v", err)
		s.AddMessage(core.Message{
			Type:    core.CommandErrorResultMessage,
			Content: fmt.Sprintf("Failed to reload context before generation:\n%v", err),
		})
		return core.Event{Type: core.MessagesUpdated}
	}

	prompt := s.GetPromptForTokenCount()
	messages := s.GetMessages()

	// Collect image paths from recent messages that precede the current user prompt.
	var imgPaths []string
	// Iterate backwards from the message before the last one (which is the user prompt).
	for i := len(messages) - 2; i >= 0; i-- {
		msg := messages[i]
		if msg.Type == core.ImageMessage {
			imgPaths = append(imgPaths, msg.Content)
		} else if msg.Type == core.UserMessage || msg.Type == core.AIMessage {
			// Stop when we hit the previous conversation turn.
			break
		}
	}
	// Reverse the slice to maintain the original order of images.
	for i, j := 0, len(imgPaths)-1; i < j; i, j = i+1, j-1 {
		imgPaths[i], imgPaths[j] = imgPaths[j], imgPaths[i]
	}

	// Convert relative image paths to absolute paths for the generation server.
	if len(imgPaths) > 0 {
		repoRoot, err := utils.FindRepoRoot()
		if err != nil {
			log.Printf("Error finding repo root for image paths: %v", err)
			s.AddMessage(core.Message{
				Type:    core.CommandErrorResultMessage,
				Content: fmt.Sprintf("Failed to resolve image paths:\n%v", err),
			})
			return core.Event{Type: core.MessagesUpdated}
		}
		for i, p := range imgPaths {
			imgPaths[i] = filepath.Join(repoRoot, p)
		}
	}

	streamChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	s.SetCancelGeneration(cancel)
	go s.GetGenerator().GenerateTask(ctx, prompt, imgPaths, streamChan, generationConfig)

	s.AddMessage(core.Message{Type: core.AIMessage, Content: ""}) // Placeholder for AI

	return core.Event{
		Type: core.GenerationStarted,
		Data: streamChan,
	}
}
