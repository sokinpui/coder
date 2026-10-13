package commands

import (
	"context"
	"fmt"

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

	_, err := s.Compact(context.Background())
	if err != nil {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: fmt.Sprintf("Compaction failed: %v", err),
		}, false
	}

	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: "Conversation successfully compacted into checkpoint.",
	}, true
}
