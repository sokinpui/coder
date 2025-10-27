package commands

import (
	"fmt"
	"strings"
)

func init() {
	registerCommand("exclude", excludeCmd, nil)
}

func excludeCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)
	cfg := s.GetConfig()

	if len(paths) == 0 {
		cfg.Sources.Exclusions = []string{}
		if err := s.LoadContext(); err != nil {
			msg := fmt.Sprintf("Project source exclusions cleared, but failed to reload context: %v", err)
			return CommandOutput{Type: CommandResultString, Payload: msg}, false
		}
		return CommandOutput{Type: CommandResultString, Payload: "Project source exclusions cleared."}, true
	}

	pathsToModify := make(map[string]struct{})
	for _, p := range paths {
		pathsToModify[p] = struct{}{}
	}

	// Remove from Dirs
	newDirs := make([]string, 0, len(cfg.Sources.Dirs))
	for _, d := range cfg.Sources.Dirs {
		if _, found := pathsToModify[d]; !found {
			newDirs = append(newDirs, d)
		}
	}
	cfg.Sources.Dirs = newDirs

	// Remove from Files
	newFiles := make([]string, 0, len(cfg.Sources.Files))
	for _, f := range cfg.Sources.Files {
		if _, found := pathsToModify[f]; !found {
			newFiles = append(newFiles, f)
		}
	}
	cfg.Sources.Files = newFiles

	// Add to Exclusions
	exclusionLookup := make(map[string]struct{})
	for _, e := range cfg.Sources.Exclusions {
		exclusionLookup[e] = struct{}{}
	}
	for _, p := range paths {
		if _, exists := exclusionLookup[p]; !exists {
			cfg.Sources.Exclusions = append(cfg.Sources.Exclusions, p)
			exclusionLookup[p] = struct{}{}
		}
	}

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	var payload strings.Builder
	payload.WriteString("Project source updated.")

	summary := formatSourceSummary(&cfg.Sources)
	if summary != "" {
		payload.WriteString("\n")
		payload.WriteString(summary)
	}

	return CommandOutput{Type: CommandResultString, Payload: payload.String()}, true
}
