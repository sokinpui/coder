package commands

import (
	"coder/internal/config"
	"coder/internal/core"
)

func init() {
	registerCommand("new", newCmd, nil)
}

func newCmd(args string, messages []core.Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	return CommandOutput{Type: CommandResultNewSession}, true
}
