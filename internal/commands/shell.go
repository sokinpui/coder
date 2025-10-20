package commands

import (
	"fmt"
	"os/exec"
	"strings"
)

func init() {
	registerCommand("shell", shellCmd, nil)
}

func shellCmd(args string, s SessionController) (CommandOutput, bool) {
	if args == "" {
		return CommandOutput{Type: CommandResultString, Payload: "Usage: :shell <command>"}, false
	}

	// If the arguments are quoted, unquote them before passing to the shell.
	// This allows users to use pipes and other special characters inside the shell command
	// without them being interpreted as coder command pipes.
	if len(args) >= 2 && ((args[0] == '\'' && args[len(args)-1] == '\'') || (args[0] == '"' && args[len(args)-1] == '"')) {
		args = args[1 : len(args)-1]
	}

	cmd := exec.Command("sh", "-c", args)
	output, err := cmd.CombinedOutput()

	result := string(output)

	if err != nil {
		errorMsg := fmt.Sprintf("Command failed: %v", err)
		result = fmt.Sprintf("%s\n%s", errorMsg, result)
		return CommandOutput{Type: CommandResultString, Payload: strings.TrimSpace(result)}, false
	}

	return CommandOutput{Type: CommandResultString, Payload: strings.TrimSpace(result)}, true
}
