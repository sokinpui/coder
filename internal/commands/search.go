package commands

func init() {
	registerCommand("search", searchCmd, nil)
}

func searchCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultSearchMode, Payload: args}, true
}
