package commands

import (
	"fmt"

	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("coding", codingCmd, "switch conversation mode to coding", nil)
}

func codingCmd(args string, s SessionController) (CommandOutput, bool) {
	if err := s.SetMode("coding"); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Failed to switch mode: %v", err)}, false
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: "Switched mode to: coding"}, true
}
