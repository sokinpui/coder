package modes

import "coder/internal/core"

// ModeStrategy defines the behavior for different application modes.
type ModeStrategy interface {
	// GetRolePrompt returns the role-specific part of the prompt.
	GetRolePrompt() string

	// LoadContext loads mode-specific context like system instructions,
	// related documents, and project source code.
	LoadContext() (systemInstructions, relatedDocuments, projectSourceCode string, err error)

	// ProcessAIResponse is a hook to perform actions after an AI response is received.
	// It's primarily used by AgentMode to execute tool calls.
	ProcessAIResponse(s SessionController) core.Event
}
