package commands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

var availableModes = []string{"coding", "chat"}

func init() {
	registerCommand("mode", modeCmd, "switch conversation mode (coding/chat)", modeArgumentCompleter)
}

func modeArgumentCompleter(cfg *config.Config, prefix string) []string {
	return availableModes
}

func modeCmd(args string, s SessionController) (CommandOutput, bool) {
	args = strings.TrimSpace(args)
	if args == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "Current mode: %s\n", s.GetMode())
		fmt.Fprintln(&b, "Available modes:")
		for _, m := range availableModes {
			fmt.Fprintf(&b, "- %s\n", m)
		}
		fmt.Fprint(&b, "Usage: /mode <mode_name>")
		return CommandOutput{Type: types.MessagesUpdated, Payload: b.String()}, true
	}

	if slices.Contains(availableModes, args) {
		if err := s.SetMode(args); err != nil {
			return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Failed to switch mode: %v", err)}, false
		}
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Switched mode to: %s", args)}, true
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error: mode '%s' not found. Available modes: %s", args, strings.Join(availableModes, ", "))}, false
}
