package commands

import (
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("compact", compactCmd, "compact conversation history into a structured checkpoint", nil)
}

func compactCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapCompact) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: compact"}, false
	}

	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: "Interactive command: use in TUI mode or session/compact RPC.",
	}, true
}
