package commands

func init() {
	registerCommand("chat", chatCmd, nil)
}

func chatCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultChatMode}, true
}
