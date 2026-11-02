package commands

import (
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("list", listCmd, "list context summary", nil)
}

func listCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapContextFiles) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: list"}, false
	}

	ctxCtrl, ok := s.(engine.ContextController)
	if !ok {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No project source files are in current context.", IsContext: true}, true
	}

	allFiles := ctxCtrl.GetContextFiles()

	if len(allFiles) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No project source files are in current context.", IsContext: true}, true
	}

	overview := formatFileListSummary(allFiles)
	summary := "Current project context:\n" + overview

	return CommandOutput{Type: types.MessagesUpdated, Payload: summary, IsContext: true}, true
}
