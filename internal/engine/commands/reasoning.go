package commands

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
)

var standardReasoningEfforts = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

func init() {
	registerCommand("reasoning", reasoningCmd, "view or switch reasoning effort / thinking level", reasoningArgumentCompleter)
	registerCommand("thinking", reasoningCmd, "view or switch reasoning effort / thinking level (alias for /reasoning)", reasoningArgumentCompleter)
}

func reasoningArgumentCompleter(s SessionController, prefix string) []string {
	return standardReasoningEfforts
}

func reasoningCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapReasoningSwitch) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: reasoning"}, false
	}

	target := strings.TrimSpace(args)
	if target == "" {
		current := s.GetReasoningEffort()
		if current == "" {
			current = "(none/default)"
		}
		msg := fmt.Sprintf("Current reasoning effort: %s\nAvailable levels: %s", current, strings.Join(standardReasoningEfforts, ", "))
		return CommandOutput{Type: types.MessagesUpdated, Payload: msg}, true
	}

	effort := strings.ToLower(target)
	if effort == "off" {
		effort = "none"
	}

	s.SetReasoningEffort(effort)
	return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Switched reasoning effort to: %s", effort)}, true
}
