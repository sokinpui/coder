package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("exclude", excludeCmd, PathArgumentCompleter)
}

func excludeCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)
	cfg := s.GetConfig()

	if len(paths) == 0 {
		cfg.Context.Exclusions = []string{}
		if err := s.LoadContext(); err != nil {
			msg := fmt.Sprintf("Project source exclusions cleared, but failed to reload context: %v", err)
			return CommandOutput{Type: types.MessagesUpdated, Payload: msg}, false
		}
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Project source exclusions cleared."}, true
	}

	pathsToRemove, _ := ExpandPaths(paths)

	pathsToModify := make(map[string]struct{})
	for _, p := range pathsToRemove {
		pathsToModify[p] = struct{}{}
	}

	// Remove from Dirs
	cfg.Context.Dirs = filterPaths(cfg.Context.Dirs, pathsToModify)

	// Remove from Files
	cfg.Context.Files = filterPaths(cfg.Context.Files, pathsToModify)

	// Add to Exclusions
	cfg.Context.Exclusions = AppendUnique(cfg.Context.Exclusions, paths)

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	var payload strings.Builder
	payload.WriteString("Project source updated.")

	summary := formatContextSummary(&cfg.Context)
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
