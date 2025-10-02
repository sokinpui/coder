package modes

import (
	"coder/internal/config"
)

// NewStrategy creates a new mode strategy based on the AppMode.
func NewStrategy(mode config.AppMode) ModeStrategy {
	switch mode {
	case config.DocumentingMode:
		return &DocumentingMode{}
	case config.AgentMode:
		return &AgentMode{activeAgent: config.MainAgent}
	case config.CodingMode:
		fallthrough
	default:
		return &CodingMode{}
	}
}
