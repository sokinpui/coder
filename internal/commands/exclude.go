package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("exclude", excludeCmd, nil)
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

	var pathsToRemove []string
	for _, p := range paths {
		if strings.ContainsAny(p, "*?[]") {
			matches, err := filepath.Glob(p)
			if err == nil { // ignore glob errors, just won't remove anything
				pathsToRemove = append(pathsToRemove, matches...)
			}
		} else {
			pathsToRemove = append(pathsToRemove, p)
		}
	}

	pathsToModify := make(map[string]struct{})
	for _, p := range pathsToRemove {
		pathsToModify[p] = struct{}{}
	}

	// Remove from Dirs
	newDirs := make([]string, 0, len(cfg.Context.Dirs))
	for _, d := range cfg.Context.Dirs {
		if _, found := pathsToModify[d]; !found {
			newDirs = append(newDirs, d)
		}
	}
	cfg.Context.Dirs = newDirs

	// Remove from Files
	newFiles := make([]string, 0, len(cfg.Context.Files))
	for _, f := range cfg.Context.Files {
		if _, found := pathsToModify[f]; !found {
			newFiles = append(newFiles, f)
		}
	}
	cfg.Context.Files = newFiles

	// Add to Exclusions
	exclusionLookup := make(map[string]struct{})
	for _, e := range cfg.Context.Exclusions {
		exclusionLookup[e] = struct{}{}
	}
	for _, p := range paths {
		if _, exists := exclusionLookup[p]; !exists {
			cfg.Context.Exclusions = append(cfg.Context.Exclusions, p)
			exclusionLookup[p] = struct{}{}
		}
	}

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
