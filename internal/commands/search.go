package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("search", searchCmd, nil)
}

func searchCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.SearchModeStarted, Payload: args}, true
}
