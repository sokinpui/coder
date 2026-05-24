package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("new", newCmd, "start new coding session", nil)
}

func newCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.NewSessionStarted, Mode: "coding"}, true
}
