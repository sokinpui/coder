package server

import "coder/internal/core"

// ClientToServerMessage represents messages sent from the client to the server.
type ClientToServerMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	RequestID string      `json:"requestId,omitempty"`
}

// ServerToClientMessage represents messages sent from the server to the client.
type ServerToClientMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	RequestID string      `json:"requestId,omitempty"`
}

// MessagePayload is a common payload structure for message updates.
type MessagePayload struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	DataURL string `json:"dataURL,omitempty"`
}

// messageTypeToString converts a core.MessageType to its string representation for client consumption.
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
	case core.ImageMessage:
		return "Image"
	default:
		return "Unknown"
	}
}
