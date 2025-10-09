package commands

func init() {
	registerCommand("new", newCmd, nil)
}

func newCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultNewSession}, true
}
