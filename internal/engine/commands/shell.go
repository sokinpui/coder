package commands

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("shell", shellCmd, "run interactive terminal command or open subshell", PathArgumentCompleter)
}

func shellCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	return CommandOutput{
		Type:    types.ShellExecutionStarted,
		Payload: trimmed,
		IsShell: true,
	}, true
}
