package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("q", quitCmd, nil)
	registerCommand("quit", quitCmd, nil)
}

func quitCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.Quit}, true
}
