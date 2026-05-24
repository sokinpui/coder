package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("fzf", fzfCmd, "open command finder", nil)
}

func fzfCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.FzfModeStarted, Payload: args}, true
}
