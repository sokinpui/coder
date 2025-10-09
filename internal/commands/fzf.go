package commands

import (
	"coder/internal/config"
	"fmt"
	"strings"
)

func init() {
	registerCommand("fzf", fzfCmd, nil)
}

// fzfCmd prepares a list of commands for fzf and returns a special command result
// to be handled by the UI.
func fzfCmd(args string, s SessionController) (CommandOutput, bool) {
	var fzfInput strings.Builder

	// mode
	for _, mode := range config.AvailableAppModes {
		fzfInput.WriteString(fmt.Sprintf("mode: %s\n", mode))
	}

	// model
	for _, model := range config.AvailableModels {
		fzfInput.WriteString(fmt.Sprintf("model: %s\n", model))
	}

	return CommandOutput{Type: CommandResultFzfMode, Payload: fzfInput.String()}, true
}
