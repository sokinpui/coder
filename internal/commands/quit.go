package commands

func init() {
	registerCommand("q", quitCmd, nil)
	registerCommand("quit", quitCmd, nil)
}

func quitCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultQuit}, true
}
