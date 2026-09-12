package commands

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

var toolsOptions = []string{"on", "off"}

func init() {
	registerCommand("tools", toolsCmd, "toggle tools execution (new sessions only)", toolsArgumentCompleter)
}

func toolsArgumentCompleter(cfg *config.Config, prefix string) []string {
	return toolsOptions
}

func toolsCmd(args string, s SessionController) (CommandOutput, bool) {
	if s.HasChatHistory() {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: "Error: Tools can only be toggled in a new session with no chat history.",
		}, false
	}

	trimmed := strings.ToLower(strings.TrimSpace(args))
	targetState := !s.IsToolsEnabled()

	switch trimmed {
	case "":
	case "on", "enable", "true", "1":
		targetState = true
	case "off", "disable", "false", "0":
		targetState = false
	default:
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: "Usage: /tools [on|off]",
		}, false
	}

	s.SetToolsEnabled(targetState)

	status := "disabled"
	if targetState {
		status = "enabled"
	}
	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: fmt.Sprintf("Tools %s.", status),
	}, true
}
