package commands

func init() {
	registerCommand("tree", treeCmd, nil)
}

func treeCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultTreeMode}, true
}
