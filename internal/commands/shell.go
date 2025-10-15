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
