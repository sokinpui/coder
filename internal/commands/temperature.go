package commands

import (
	"fmt"
	"strconv"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("temperature", tempCmd, nil)
}

func tempCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	if args == "" {
		payload := fmt.Sprintf("Current temperature: %.1f\nUsage: :temp <value>", cfg.Generation.Temperature)
		return CommandOutput{Type: types.MessagesUpdated, Payload: payload}, true
	}

	temp, err := strconv.ParseFloat(args, 32)
	if err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error: invalid temperature value '%s'. Please provide a number.", args)}, false
	}

	if temp < 0.0 || temp > 2.0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Error: temperature must be between 0.0 and 2.0."}, false
	}

	cfg.Generation.Temperature = float32(temp)
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Set temperature to: %.1f", temp)}, true
}
