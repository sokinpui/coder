package server

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/generation"
	"coder/internal/source"
	"context"
	"fmt"
	"log"
)

// handleAskAI processes an "askAI" request, performing a one-shot generation.
func (c *Client) handleAskAI(payload map[string]interface{}, requestID string) {
	if requestID == "" {
		log.Println("askAI message received without a requestID")
		return
	}

	contextStr, _ := payload["context"].(string)
	question, _ := payload["question"].(string)
	historyData, _ := payload["history"].([]interface{})

	// This is a one-shot generation, so we create a temporary generator.
	// We can reuse the main session's config.
	cfg := c.session.GetConfig()
	generator, err := generation.New(cfg)
	if err != nil {
		log.Printf("failed to create generator for askAI: %v", err)
		c.send <- ServerToClientMessage{
			Type:      "error",
			Payload:   "Failed to initialize AI model for this request.",
			RequestID: requestID,
		}
		return
	}

	// The preamble is treated as a related document for the prompt.
	preamble := fmt.Sprintf("The user has highlighted the following snippet and has a question about it:\n\n```\n%s\n```", contextStr)

	// Convert history from frontend
	var messages []core.Message
	for _, h := range historyData {
		histItem, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		sender, _ := histItem["sender"].(string)
		content, _ := histItem["content"].(string)
		var msgType core.MessageType
		if sender == "User" {
			msgType = core.UserMessage
		} else if sender == "AI" {
			msgType = core.AIMessage
		} else {
			continue
		}
		messages = append(messages, core.Message{Type: msgType, Content: content})
	}

	// Add the new question
	messages = append(messages, core.Message{Type: core.UserMessage, Content: question})

	// Load project source, including markdown files, to provide full context.
	projectSource, err := source.LoadProjectSource(config.AutoMode)
	if err != nil {
		log.Printf("failed to load project source for askAI: %v", err)
		projectSource = "" // Proceed without project source on error
	}

	// Build the prompt
	prompt := core.BuildPrompt(core.AskAIRole, "", preamble, projectSource, messages)

	streamChan := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	c.askAICancels[requestID] = cancel
	c.mu.Unlock()

	go generator.GenerateTask(ctx, prompt, nil, streamChan)

	// Stream results back to client
	go func() {
		defer func() {
			cancel()
			c.mu.Lock()
			delete(c.askAICancels, requestID)
			c.mu.Unlock()
		}()
		for chunk := range streamChan {
			c.send <- ServerToClientMessage{
				Type:      "generationChunk",
				Payload:   chunk,
				RequestID: requestID,
			}
		}
		c.send <- ServerToClientMessage{Type: "generationEnd", RequestID: requestID}
	}()
}
