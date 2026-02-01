package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("chat", chatCmd, nil)
}

func chatCmd(args string, s SessionController) (CommandOutput, bool) {
	s.StartChatSession()
	return CommandOutput{Type: types.NewSessionStarted}, true
}
