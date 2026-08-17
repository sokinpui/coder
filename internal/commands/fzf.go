package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("fzf", fzfCmd, "open model switcher", nil)
}

func fzfCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.FzfModeStarted, Payload: args}, true
}
