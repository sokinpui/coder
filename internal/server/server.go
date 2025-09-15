package server

import (
	"coder/internal/config"
	"coder/internal/core"
	"coder/internal/session"
	"encoding/json"
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
	Type    string `json:"type"`
	Payload string `json:"payload"`
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
	defer c.conn.Close()

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

		go c.handleClientMessage(msg)
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

func (c *Client) handleClientMessage(msg ClientToServerMessage) {
	if msg.Type != "userInput" {
		log.Printf("unknown message type: %s", msg.Type)
		return
	}

	event := c.session.HandleInput(msg.Payload)

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
				Content: lastMsg.Content,
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

	client.send <- ServerToClientMessage{
		Type: "initialState",
		Payload: map[string]string{
			"cwd": getShortCwd(),
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
