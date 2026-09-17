package commands

import (
	"fmt"

	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("chat", chatCmd, "switch conversation mode to chat", nil)
}

func chatCmd(args string, s SessionController) (CommandOutput, bool) {
	if err := s.SetMode("chat"); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Failed to switch mode: %v", err)}, false
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Switched mode to: chat"}, true
}
