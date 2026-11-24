package commands

import (
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("btw", btwCmd, "ask a side question in an ephemeral overlay", nil)
}

func btwCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: "Interactive command: use in TUI mode.",
	}, true
}
