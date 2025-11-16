package modes

import (
	"coder/internal/config"
	"coder/internal/types"
	"coder/internal/utils"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// StartGeneration provides a default implementation for starting a generation task,
// optionally allowing a specific generation config to be used.
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

	prompt := s.GetPrompt()
	messages := s.GetMessages()

	// Collect image paths from recent messages that precede the current user prompt.
	var imgPaths []string
	// Iterate backwards from the message before the last one (which is the user prompt).
	for i := len(messages) - 2; i >= 0; i-- {
		msg := messages[i]
		if msg.Type == types.ImageMessage {
			imgPaths = append(imgPaths, msg.Content)
		} else if msg.Type == types.UserMessage || msg.Type == types.AIMessage {
			// Stop when we hit the previous conversation turn.
			break
		}
	}
	// Reverse the slice to maintain the original order of images.
	for i, j := 0, len(imgPaths)-1; i < j; i, j = i+1, j-1 {
		imgPaths[i], imgPaths[j] = imgPaths[j], imgPaths[i]
	}

	// Convert relative image paths to absolute paths for the generation server.
	var images [][]byte
	if len(imgPaths) > 0 {
		repoRoot, err := utils.FindRepoRoot()
		if err != nil {
			log.Printf("Error finding repo root for image paths: %v", err)
			s.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: fmt.Sprintf("Failed to read images: could not find repository root: %v", err),
			})
			return types.Event{Type: types.MessagesUpdated}
		}
		for _, p := range imgPaths {
			absPath := filepath.Join(repoRoot, p)
			imgBytes, err := os.ReadFile(absPath)
			if err != nil {
				log.Printf("Error reading image file %s: %v", absPath, err)
				s.AddMessages(types.Message{
					Type:    types.CommandErrorResultMessage,
					Content: fmt.Sprintf("Failed to read image file %s:\n%v", p, err),
				})
				return types.Event{Type: types.MessagesUpdated}
			}
			images = append(images, imgBytes)
		}
	}

	streamChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	s.SetCancelGeneration(cancel)
	go s.GetGenerator().GenerateTask(ctx, prompt, images, streamChan, generationConfig)

	s.AddMessages(types.Message{Type: types.AIMessage, Content: ""}) // Placeholder for AI

	return types.Event{
		Type: types.GenerationStarted,
		Data: streamChan,
	}
}
