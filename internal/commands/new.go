package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("new", newCmd, nil)
}

func newCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.NewSessionStarted, Mode: "coding"}, true
}
