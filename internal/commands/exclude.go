package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("exclude", excludeCmd, "exclude path from context", PathArgumentCompleter)
}

func excludeCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)

	if len(paths) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Exclusions command requires arguments to specify what to remove from current context."}, false
	}

	pathsToRemove, _ := ExpandPaths(paths)

	pathsToModify := make(map[string]struct{})
	for _, p := range pathsToRemove {
		pathsToModify[p] = struct{}{}
	}

	currentFiles := s.GetContextFiles()
	s.SetContextFiles(filterPaths(currentFiles, pathsToModify))

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	var payload strings.Builder
	payload.WriteString("Project source updated.")

	summary := formatFileListSummary(s.GetContextFiles())
	if summary != "" {
		payload.WriteString("\n")
		payload.WriteString(summary)
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: payload.String()}, true
}

func filterPaths(original []string, toRemove map[string]struct{}) []string {
	filtered := make([]string, 0, len(original))
	for _, p := range original {
		if _, found := toRemove[p]; !found {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
