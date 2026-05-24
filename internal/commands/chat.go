package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("chat", chatCmd, "start new chat (no context)", nil)
}

func chatCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.NewSessionStarted, Mode: "chat"}, true
}
