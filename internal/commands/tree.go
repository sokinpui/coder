package commands

import "github.com/sokinpui/coder/internal/types"

func init() {
	registerCommand("tree", treeCmd, nil)
}

func treeCmd(args string, s SessionController) (CommandOutput, bool) {
	return CommandOutput{Type: types.TreeModeStarted}, true
}
