package commands

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("term", termCmd, "run interactive terminal command or open subshell", PathArgumentCompleter)
}

func termCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	return CommandOutput{
		Type:    types.TermExecutionStarted,
		Payload: trimmed,
		IsShell: true,
	}, true
}
