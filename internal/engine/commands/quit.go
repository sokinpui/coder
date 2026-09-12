package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("quit", quitCmd, "quit application", nil)
}

func quitCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.Quit}, true
}
