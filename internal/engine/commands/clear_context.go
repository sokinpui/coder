package commands

import (
	"fmt"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("clear_context", clearContextCmd, "clear all context files and documents", nil)
}

func clearContextCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapContextFiles) {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Unknown command: clear_context"}, false
	}

	ctxCtrl, ok := s.(engine.ContextController)
	if !ok {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Context management not supported in this session"}, false
	}

	ctxCtrl.SetContextFiles([]string{})
	ctxCtrl.SetContextDocuments([]string{})
	ctxCtrl.ClearAllDocumentMessages()

	if err := ctxCtrl.LoadContext(); err != nil {
		msg := fmt.Sprintf("Project context cleared, but failed to reload context: %v", err)
		return CommandOutput{Type: types.MessagesUpdated, Payload: msg}, false
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: "Project context cleared.", IsContext: true}, true
}
