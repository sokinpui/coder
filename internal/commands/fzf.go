package commands

func init() {
	registerCommand("fzf", fzfCmd, nil)
}

func fzfCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultFzfMode, Payload: args}, true
}
