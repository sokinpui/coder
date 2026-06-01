package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/itf"
	"strings"
)

func init() {
	registerCommand("undo", undoCmd, "undo last file changes", nil)
}

func undoCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.HasAppliedChanges() {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No changes have been applied in this session to undo."}, false
	}

	itfCfg := &itf.Config{Undo: true}
	app, err := itf.NewApp(itfCfg)
	if err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Failed to initialize itf: " + err.Error()}, false
	}

	summary, err := app.Execute()
	if err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Error undoing changes: " + err.Error()}, false
	}

	if summary.Message == "No undo" {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No changes to undo."}, true
	}

	// Update context paths based on what was undone
	currentFiles := s.GetContextFiles()
	contextUpdated := false

	// If we undone a creation, the file is deleted
	if len(summary.Deleted) > 0 {
		toRemove := make(map[string]struct{})
		for _, p := range summary.Deleted {
			toRemove[p] = struct{}{}
		}
		currentFiles = filterPaths(currentFiles, toRemove)
		contextUpdated = true
	}

	// If we undone a deletion, the file is created (restored)
	if len(summary.Created) > 0 {
		currentFiles = AppendUnique(currentFiles, summary.Created)
		contextUpdated = true
	}

	// If we undone a rename
	if len(summary.Renamed) > 0 {
		toRemove := make(map[string]struct{})
		var toAdd []string
		for _, entry := range summary.Renamed {
			parts := strings.Split(entry, " -> ")
			if len(parts) == 2 {
				// In undo, "old -> new" means it was renamed from 'new' back to 'old'
				// But summary.Renamed is already relativized and formatted by itf as "orig -> back"
				toRemove[parts[0]] = struct{}{}
				toAdd = append(toAdd, parts[1])
			}
		}
		currentFiles = filterPaths(currentFiles, toRemove)
		currentFiles = AppendUnique(currentFiles, toAdd)
		contextUpdated = true
	}

	if contextUpdated {
		s.SetContextFiles(currentFiles)
		_ = s.LoadContext()
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: itf.FormatSummary(summary)}, true
}
