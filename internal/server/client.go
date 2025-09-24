package server

import (
	"coder/internal/session"
	"coder/internal/token"
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a single connected WebSocket client.
type Client struct {
	conn         *websocket.Conn
	session      *session.Session
	send         chan ServerToClientMessage
	askAICancels map[string]context.CancelFunc
	mu           sync.Mutex
}

// readPump pumps messages from the websocket connection to the client.
func (c *Client) readPump() {
	defer func() {
		if err := c.session.SaveConversation(); err != nil {
			log.Printf("Error saving conversation on disconnect: %v", err)
		}
		c.mu.Lock()
		for _, cancel := range c.askAICancels {
			cancel()
		}
		c.mu.Unlock()
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg ClientToServerMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshalling message: %v", err)
			continue
		}

		switch msg.Type {
		case "userInput":
			if payload, ok := msg.Payload.(string); ok {
				c.handleUserInput(payload)
			}
		case "imageUpload":
			if payload, ok := msg.Payload.(string); ok {
				c.handleImageUpload(payload)
			}
		case "cancelGeneration":
			log.Println("Received cancel generation request")
			c.session.CancelGeneration()
		case "regenerateFrom":
			if index, ok := msg.Payload.(float64); ok { // JSON numbers are float64
				c.handleRegenerate(int(index))
			}
		case "applyItf":
			if content, ok := msg.Payload.(string); ok {
				c.handleApplyItf(content)
			}
		case "editMessage":
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				index, indexOk := payload["index"].(float64) // JSON numbers are float64
				content, contentOk := payload["content"].(string)
				if indexOk && contentOk {
					c.handleEditMessage(int(index), content)
				}
			}
		case "branchFrom":
			if index, ok := msg.Payload.(float64); ok { // JSON numbers are float64
				c.handleBranchFrom(int(index))
			}
		case "deleteMessage":
			if index, ok := msg.Payload.(float64); ok { // JSON numbers are float64
				c.handleDeleteMessage(int(index))
			}
		case "listHistory":
			c.handleListHistory()
		case "loadConversation":
			if filename, ok := msg.Payload.(string); ok {
				c.handleLoadConversation(filename)
			}
		case "getSourceTree":
			c.handleGetSourceTree()
		case "getFileContent":
			if path, ok := msg.Payload.(string); ok {
				c.handleGetFileContent(path)
			}
		case "getGitGraphLog":
			c.handleGetGitGraphLog()
		case "getCommitDiff":
			if hash, ok := msg.Payload.(string); ok {
				c.handleGetCommitDiff(hash)
			}
		case "askAI":
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				c.handleAskAI(payload, msg.RequestID)
			}
		default:
			log.Printf("unknown message type: %s", msg.Type)
		}
	}
}

// writePump pumps messages from the client to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("error writing json: %v", err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// updateClientState sends the current session mode, model, and token count to the client.
func (c *Client) updateClientState() {
	tokenCount := token.CountTokens(c.session.GetPromptForTokenCount())
	c.send <- ServerToClientMessage{
		Type: "stateUpdate",
		Payload: map[string]interface{}{
			"mode":       string(c.session.GetConfig().AppMode),
			"model":      c.session.GetConfig().Generation.ModelCode,
			"tokenCount": tokenCount,
		},
	}
}
