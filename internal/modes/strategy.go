package modes

import (
	"coder/internal/types"
	"coder/internal/config"
)

// ModeStrategy defines the behavior for different application modes.
type ModeStrategy interface {
	// GetRolePrompt returns the role-specific part of the prompt.
	GetRolePrompt() string

	// LoadSourceCode loads and stores mode-specific context.
	LoadSourceCode(cfg *config.Config) error

	// StartGeneration prepares and begins a new AI generation task.
	StartGeneration(s SessionController) types.Event

	// BuildPrompt constructs the full prompt for the model.
	BuildPrompt(messages []types.Message) string
}
