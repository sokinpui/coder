package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("new", newCmd, "start new coding session", nil)
}

func newCmd(args string, s SessionController) (CommandOutput, bool) {
	mode := "coder"
	if s != nil && s.GetMode() != "" {
		mode = s.GetMode()
	}
	return CommandOutput{Type: types.NewSessionStarted, Mode: mode}, true
}
