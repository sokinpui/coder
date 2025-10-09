package commands

import (
	"coder/internal/config"
	"fmt"
	"strings"
)

func init() {
	registerCommand("model", modelCmd, modelArgumentCompleter)
}

func modelArgumentCompleter(cfg *config.Config) []string {
	return config.AvailableModels
}

func modelCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	if args == "" {
		var b strings.Builder
		fmt.Fprintf(&b, "Current model: %s\n", cfg.Generation.ModelCode)
		fmt.Fprintln(&b, "Available models:")
		for _, m := range config.AvailableModels {
			fmt.Fprintf(&b, "- %s\n", m)
		}
		fmt.Fprint(&b, "Usage: :model <model_name>")
		return CommandOutput{Type: CommandResultString, Payload: b.String()}, true
	}

	for _, m := range config.AvailableModels {
		if m == args {
			cfg.Generation.ModelCode = args
			return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Switched model to: %s", args)}, true
		}
	}

	return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error: model '%s' not found. Use ':model' to see available models.", args)}, false
}
