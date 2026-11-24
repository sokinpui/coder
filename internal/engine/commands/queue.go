package commands

import (
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("queue", queueCmd, "view and manage queued prompts", nil)
}

func queueCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapToolLoop) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: queue"}, false
	}

	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: "Interactive command: use in TUI mode.",
	}, true
}
