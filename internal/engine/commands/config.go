package commands

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
	"gopkg.in/yaml.v3"
)

func init() {
	registerCommand("config", configCmd, "show or reload configuration", configArgumentCompleter)
}

func configArgumentCompleter(cfg *config.Config, prefix string) []string {
	return []string{"reload"}
}

func configCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "reload" {
		if err := s.ReloadConfig(); err != nil {
			return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Failed to reload config: %v", err)}, false
		}
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Configuration reloaded successfully."}, true
	}

	data, err := yaml.Marshal(s.GetConfig())
	if err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error marshaling config: %v", err)}, false
	}
	return CommandOutput{Type: types.MessagesUpdated, Payload: string(data)}, true
}
