package commands

import (
	"fmt"
	"slices"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("model", modelCmd, "switch generation model", modelArgumentCompleter)
}

func modelArgumentCompleter(cfg *config.Config, prefix string) []string {
	return cfg.AvailableModels
}

func modelCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	if args == "" {
		return CommandOutput{Type: types.FzfModeStarted, Payload: ""}, true
	}

	if slices.Contains(cfg.AvailableModels, args) {
		cfg.Generation.ModelCode = args
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Switched model to: %s", args)}, true
	}

	return CommandOutput{Type: types.FzfModeStarted, Payload: args}, true
}
