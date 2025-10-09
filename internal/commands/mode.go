package commands

import (
	"coder/internal/config"
	"coder/internal/core"
	"fmt"
	"strings"
)

func init() {
	registerCommand("mode", modeCmd, modeArgumentCompleter)
}

func modeArgumentCompleter(cfg *config.Config) []string {
	modes := make([]string, len(config.AvailableAppModes))
	for i, m := range config.AvailableAppModes {
		modes[i] = string(m)
	}
	return modes
}

func modeCmd(args string, messages []core.Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	if args == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "Current mode: %s\n", cfg.AppMode)
		fmt.Fprintln(&b, "Available modes:")
		for _, m := range config.AvailableAppModes {
			fmt.Fprintf(&b, "- %s\n", m)
		}
		fmt.Fprint(&b, "Usage: :mode <mode_name>")
		return CommandOutput{Type: CommandResultString, Payload: b.String()}, true
	}

	requestedMode := config.AppMode(args)
	for _, m := range config.AvailableAppModes {
		if m == requestedMode {
			if err := sess.SetMode(requestedMode); err != nil {
				return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error switching mode: %v", err)}, false
			}
			return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Switched mode to: %s", args)}, true
		}
	}

	return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error: mode '%s' not found. Use ':mode' to see available modes.", args)}, false
}
