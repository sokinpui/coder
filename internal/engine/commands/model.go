package commands

import (
	"fmt"
	"strings"

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
		msg := fmt.Sprintf("Current model: %s", cfg.Generation.ModelCode)
		if len(cfg.AvailableModels) > 0 {
			msg += fmt.Sprintf("\nAvailable models: %s", strings.Join(cfg.AvailableModels, ", "))
		}
		return CommandOutput{Type: types.MessagesUpdated, Payload: msg}, true
	}

	s.SetModel(args)
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Switched model to: %s", args)}, true
}
