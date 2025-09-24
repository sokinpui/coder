package server

import (
	"coder/internal/core"
	"coder/internal/session"
	"coder/internal/utils"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"coder/internal/token"
)

// handleUserInput processes a user's text input.
func (c *Client) handleUserInput(payload string) {
	isFirstUserMessage := !c.session.IsTitleGenerated() && !strings.HasPrefix(payload, ":")

	if isFirstUserMessage {
		go func() {
			title := c.session.GenerateTitle(context.Background(), payload)
			c.send <- ServerToClientMessage{
				Type:    "titleUpdate",
				Payload: title,
			}
		}()
	}

	event := c.session.HandleInput(payload)

	c.updateClientState()

	switch event.Type {
	case session.NoOp:
		return

	case session.MessagesUpdated:
		messages := c.session.GetMessages()
		if len(messages) == 0 {
			return
		}
		lastMsg := messages[len(messages)-1]
		c.send <- ServerToClientMessage{
			Type: "messageUpdate",
			Payload: MessagePayload{
				Type:    messageTypeToString(lastMsg.Type),
				Content: utils.StripAnsi(lastMsg.Content),
				DataURL: lastMsg.DataURL,
			},
		}

	case session.NewSessionStarted:
		c.send <- ServerToClientMessage{Type: "newSession"}

	case session.GenerationStarted:
		streamChan, ok := event.Data.(chan string)
		if !ok {
			return
		}
		go func() {
			for chunk := range streamChan {
				if chunk != "" {
					c.send <- ServerToClientMessage{Type: "generationChunk", Payload: chunk}
				}
			}
			c.send <- ServerToClientMessage{Type: "generationEnd"}
		}()
	}
}

// handleRegenerate truncates the message history to a specific user message and initiates a new generation.
func (c *Client) handleRegenerate(userMessageIndex int) {
	event := c.session.RegenerateFrom(userMessageIndex)

	c.updateClientState()

	switch event.Type {
	case session.MessagesUpdated: // This is for errors
		messages := c.session.GetMessages()
		if len(messages) == 0 {
			return
		}
		lastMsg := messages[len(messages)-1]
		c.send <- ServerToClientMessage{
			Type: "messageUpdate",
			Payload: MessagePayload{
				Type:    messageTypeToString(lastMsg.Type),
				Content: utils.StripAnsi(lastMsg.Content),
				DataURL: lastMsg.DataURL,
			},
		}

	case session.GenerationStarted:
		// Tell the client to truncate its message list.
		// The session now has `userMessageIndex + 1` messages.
		c.send <- ServerToClientMessage{
			Type:    "truncateMessages",
			Payload: userMessageIndex + 1,
		}

		streamChan, ok := event.Data.(chan string)
		if !ok {
			return
		}
		go func() {
			for chunk := range streamChan {
				c.send <- ServerToClientMessage{Type: "generationChunk", Payload: chunk}
			}
			c.send <- ServerToClientMessage{Type: "generationEnd"}
		}()
	}
}

// handleApplyItf executes the 'itf' command with the given content.
func (c *Client) handleApplyItf(content string) {
	result, success := core.ExecuteItf(content, "")

	cmdMsg := core.Message{
		Type:    core.CommandMessage,
		Content: ":itf (from apply button)",
	}
	c.session.AddMessage(cmdMsg)
	c.send <- ServerToClientMessage{
		Type: "messageUpdate",
		Payload: MessagePayload{
			Type:    messageTypeToString(cmdMsg.Type),
			Content: utils.StripAnsi(cmdMsg.Content),
		},
	}

	var resultMsgType core.MessageType
	if success {
		resultMsgType = core.CommandResultMessage
	} else {
		resultMsgType = core.CommandErrorResultMessage
	}
	resultMsg := core.Message{
		Type:    resultMsgType,
		Content: result,
	}
	c.session.AddMessage(resultMsg)
	c.send <- ServerToClientMessage{
		Type:    "messageUpdate",
		Payload: MessagePayload{Type: messageTypeToString(resultMsg.Type), Content: utils.StripAnsi(resultMsg.Content)},
	}
}

// handleEditMessage updates the content of a specific message in the session.
func (c *Client) handleEditMessage(index int, newContent string) {
	err := c.session.EditMessage(index, newContent)
	if err != nil {
		log.Printf("Error editing message: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: fmt.Sprintf("Failed to edit message: %v", err),
		}
	}
	// No success message needed, client updates optimistically.
}

// handleBranchFrom creates a new session by branching from a specific message index.
func (c *Client) handleBranchFrom(endMessageIndex int) {
	newSess, err := c.session.Branch(endMessageIndex)
	if err != nil {
		log.Printf("Error branching session: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: fmt.Sprintf("Failed to branch session: %v", err),
		}
		return
	}

	c.session = newSess

	// Tell the client to truncate its message list.
	// The new session has `endMessageIndex + 1` messages.
	c.send <- ServerToClientMessage{
		Type:    "truncateMessages",
		Payload: endMessageIndex + 1,
	}

	// Send updated state like token count
	c.updateClientState()
}

// handleDeleteMessage removes a message from the session.
func (c *Client) handleDeleteMessage(index int) {
	// The session's DeleteMessages method takes a slice of indices.
	c.session.DeleteMessages([]int{index})
}

// handleListHistory retrieves and sends the list of conversation history items.
func (c *Client) handleListHistory() {
	items, err := c.session.GetHistoryManager().ListConversations()
	if err != nil {
		log.Printf("Error listing history: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: "Failed to list conversation history.",
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type:    "historyList",
		Payload: items,
	}
}

// handleLoadConversation loads a specific conversation from history into the current session.
func (c *Client) handleLoadConversation(filename string) {
	err := c.session.LoadConversation(filename)
	if err != nil {
		log.Printf("Error loading conversation: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: fmt.Sprintf("Failed to load conversation: %v", err),
		}
		return
	}

	messages := c.session.GetMessages()

	payloadMessages := make([]MessagePayload, len(messages))
	for i, msg := range messages {
		dataURL := msg.DataURL
		if msg.Type == core.ImageMessage && dataURL == "" {
			repoRoot, err := utils.FindRepoRoot()
			if err == nil {
				imagePath := filepath.Join(repoRoot, msg.Content)
				imageData, err := os.ReadFile(imagePath)
				if err == nil {
					mimeType := http.DetectContentType(imageData)
					dataURL = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(imageData)
				} else {
					log.Printf("Failed to read image file %s for history: %v", imagePath, err)
				}
			}
		}

		payloadMessages[i] = MessagePayload{
			Type:    messageTypeToString(msg.Type),
			Content: utils.StripAnsi(msg.Content),
			DataURL: dataURL,
		}
	}

	c.send <- ServerToClientMessage{
		Type: "sessionLoaded",
		Payload: map[string]interface{}{
			"messages":   payloadMessages,
			"title":      c.session.GetTitle(),
			"mode":       string(c.session.GetConfig().AppMode),
			"model":      c.session.GetConfig().Generation.ModelCode,
			"tokenCount": token.CountTokens(c.session.GetPromptForTokenCount()),
		},
	}
}
