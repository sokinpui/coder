package commands

func init() {
	registerCommand("new", newCmd, nil)
}

func newCmd(args string, s Session) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultNewSession}, true
}
