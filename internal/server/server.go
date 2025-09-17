package server

import (
	"coder/internal/config"
	"coder/internal/git"
	"coder/internal/core"
	"coder/internal/files"
	"coder/internal/session"
	"coder/internal/token"
	"coder/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type ClientToServerMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ServerToClientMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func getShortCwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown directory"
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(wd, home) {
		return "~" + strings.TrimPrefix(wd, home)
	}
	return wd
}

type MessagePayload struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

type Client struct {
	conn    *websocket.Conn
	session *session.Session
	send    chan ServerToClientMessage
}

func (c *Client) readPump() {
	defer func() {
		if err := c.session.SaveConversation(); err != nil {
			log.Printf("Error saving conversation on disconnect: %v", err)
		}
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
		case "getGitLog":
			c.handleGetGitLog()
		default:
			log.Printf("unknown message type: %s", msg.Type)
		}
	}
}

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

	tokenCount := token.CountTokens(c.session.GetPromptForTokenCount())
	c.send <- ServerToClientMessage{
		Type: "stateUpdate",
		Payload: map[string]interface{}{
			"mode":       string(c.session.GetConfig().AppMode),
			"model":      c.session.GetConfig().Generation.ModelCode,
			"tokenCount": tokenCount,
		},
	}

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

func (c *Client) handleRegenerate(userMessageIndex int) {
	event := c.session.RegenerateFrom(userMessageIndex)

	tokenCount := token.CountTokens(c.session.GetPromptForTokenCount())
	c.send <- ServerToClientMessage{
		Type: "stateUpdate",
		Payload: map[string]interface{}{
			"mode":       string(c.session.GetConfig().AppMode),
			"model":      c.session.GetConfig().Generation.ModelCode,
			"tokenCount": tokenCount,
		},
	}

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

func (c *Client) handleDeleteMessage(index int) {
	// The session's DeleteMessages method takes a slice of indices.
	c.session.DeleteMessages([]int{index})
}

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
		payloadMessages[i] = MessagePayload{
			Type:    messageTypeToString(msg.Type),
			Content: utils.StripAnsi(msg.Content),
		}
	}

	tokenCount := token.CountTokens(c.session.GetPromptForTokenCount())

	c.send <- ServerToClientMessage{
		Type: "sessionLoaded",
		Payload: map[string]interface{}{
			"messages":   payloadMessages,
			"title":      c.session.GetTitle(),
			"mode":       string(c.session.GetConfig().AppMode),
			"model":      c.session.GetConfig().Generation.ModelCode,
			"tokenCount": tokenCount,
		},
	}
}

func (c *Client) handleGetSourceTree() {
	tree, err := files.GetFileTree()
	if err != nil {
		log.Printf("Error getting source tree: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: "Failed to get project file tree.",
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type:    "sourceTree",
		Payload: tree,
	}
}

func (c *Client) handleGetFileContent(path string) {
	content, err := files.GetFileContent(path)
	if err != nil {
		log.Printf("Error getting file content for %s: %v", path, err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: fmt.Sprintf("Failed to get content for file: %s", path),
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type: "fileContent",
		Payload: map[string]string{
			"path":    path,
			"content": content,
		},
	}
}

func (c *Client) handleGetGitLog() {
	logEntries, err := git.GetLog()
	if err != nil {
		log.Printf("Error getting git log: %v", err)
		c.send <- ServerToClientMessage{
			Type:    "error",
			Payload: "Failed to get git log.",
		}
		return
	}
	c.send <- ServerToClientMessage{
		Type:    "gitLog",
		Payload: logEntries,
	}
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	cfg := config.Default()
	sess, err := session.New(cfg)
	if err != nil {
		log.Printf("failed to create session: %v", err)
		conn.Close()
		return
	}

	if err := sess.LoadContext(); err != nil {
		log.Printf("failed to load context: %v", err)
		msg, _ := json.Marshal(ServerToClientMessage{
			Type:    "error",
			Payload: "Failed to load repository context.",
		})
		conn.WriteMessage(websocket.TextMessage, msg)
	}

	client := &Client{
		conn:    conn,
		session: sess,
		send:    make(chan ServerToClientMessage, 256),
	}

	modes := make([]string, len(config.AvailableAppModes))
	for i, m := range config.AvailableAppModes {
		modes[i] = string(m)
	}

	initialTokenCount := token.CountTokens(sess.GetInitialPromptForTokenCount())

	client.send <- ServerToClientMessage{
		Type: "initialState",
		Payload: map[string]interface{}{
			"cwd":             getShortCwd(),
			"title":           sess.GetTitle(),
			"mode":            string(cfg.AppMode),
			"model":           cfg.Generation.ModelCode,
			"tokenCount":      initialTokenCount,
			"availableModes":  modes,
			"availableModels": config.AvailableModels,
		},
	}

	log.Println("Client connected")
	go client.writePump()
	client.readPump()
	log.Println("Client disconnected")
}

func messageTypeToString(msgType core.MessageType) string {
	switch msgType {
	case core.UserMessage:
		return "User"
	case core.AIMessage:
		return "AI"
	case core.ActionMessage, core.CommandMessage:
		return "Command"
	case core.ActionResultMessage, core.CommandResultMessage:
		return "Result"
	case core.ActionErrorResultMessage, core.CommandErrorResultMessage:
		return "Error"
	case core.InitMessage, core.DirectoryMessage:
		return "System"
	default:
		return "Unknown"
	}
}
