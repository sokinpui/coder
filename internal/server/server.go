package server

import (
	"coder/internal/config"
	"coder/internal/session"
	"coder/internal/token"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

// HandleConnections handles new WebSocket connections.
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
		conn:         conn,
		session:      sess,
		send:         make(chan ServerToClientMessage, 256),
		askAICancels: make(map[string]context.CancelFunc),
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
