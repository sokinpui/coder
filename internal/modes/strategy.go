package modes

import (
	"coder/internal/config"
	"coder/internal/core"
)

// ModeStrategy defines the behavior for different application modes.
type ModeStrategy interface {
	// GetRolePrompt returns the role-specific part of the prompt.
	GetRolePrompt() string

	// LoadSourceCode loads and stores mode-specific context.
	LoadSourceCode(cfg *config.Config) error

	// StartGeneration prepares and begins a new AI generation task.
	StartGeneration(s SessionController) core.Event

	// BuildPrompt constructs the full prompt for the model.
	BuildPrompt(messages []core.Message) string
}
