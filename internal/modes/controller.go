package modes

import "coder/internal/core"

// SessionController defines the parts of a session that a mode strategy can control.
// It is implemented by session.Session.
type SessionController interface {
	GetMessages() []core.Message
	AddMessage(msg core.Message)
	StartGeneration() core.Event
}
