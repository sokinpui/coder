package commands

import (
	"coder/internal/config"
	"coder/internal/core"
	"os/exec"
	"strings"
)

func init() {
	registerCommand("itf", itfCmd, nil)
}

// ExecuteItf runs the 'itf' command with the given content as stdin.
func ExecuteItf(content string, args string) (string, bool) {
	allArgs := append([]string{"--no-animation"}, strings.Fields(args)...)
	cmd := exec.Command("itf", allArgs...)
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Error executing itf: " + err.Error() + "\n" + string(output), false
	}
	return string(output), true
}

func itfCmd(args string, messages []core.Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	var lastAIResponse string
	found := false
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == core.AIMessage {
			lastAIResponse = messages[i].Content
			found = true
			break
		}
	}

	if !found {
		return CommandOutput{Type: CommandResultString, Payload: "No AI response found to pipe to itf."}, false
	}

	result, success := ExecuteItf(lastAIResponse, args)
	return CommandOutput{Type: CommandResultString, Payload: result}, success
}
