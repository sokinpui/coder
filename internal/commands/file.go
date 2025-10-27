package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	registerCommand("file", fileCmd, nil)
}

func fileCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)
	cfg := s.GetConfig()

	if len(paths) == 0 {
		cfg.Context.Dirs = []string{}
		cfg.Context.Files = []string{}
		if err := s.LoadContext(); err != nil {
			msg := fmt.Sprintf("Project context cleared, but failed to reload context: %v", err)
			return CommandOutput{Type: CommandResultString, Payload: msg}, false
		}
		return CommandOutput{Type: CommandResultString, Payload: "Project context cleared. The next prompt will not include any project source code."}, true
	}

	var files []string
	var dirs []string
	var invalidPaths []string

	var expandedPaths []string
	for _, p := range paths {
		if strings.ContainsAny(p, "*?[]") {
			matches, err := filepath.Glob(p)
			if err != nil {
				invalidPaths = append(invalidPaths, fmt.Sprintf("%s (glob error: %v)", p, err))
				continue
			}
			if len(matches) == 0 {
				invalidPaths = append(invalidPaths, p)
			}
			expandedPaths = append(expandedPaths, matches...)
		} else {
			expandedPaths = append(expandedPaths, p)
		}
	}

	for _, p := range expandedPaths {
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				invalidPaths = append(invalidPaths, p)
				continue
			}
			return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error accessing path %s: %v", p, err)}, false
		}
		if info.IsDir() {
			dirs = append(dirs, p)
		} else {
			files = append(files, p)
		}
	}

	// De-duplicate and append directories
	dirLookup := make(map[string]struct{})
	for _, d := range cfg.Context.Dirs {
		dirLookup[d] = struct{}{}
	}
	for _, d := range dirs {
		if _, exists := dirLookup[d]; !exists {
			cfg.Context.Dirs = append(cfg.Context.Dirs, d)
			dirLookup[d] = struct{}{}
		}
	}

	// De-duplicate and append files
	fileLookup := make(map[string]struct{})
	for _, f := range cfg.Context.Files {
		fileLookup[f] = struct{}{}
	}
	for _, f := range files {
		if _, exists := fileLookup[f]; !exists {
			cfg.Context.Files = append(cfg.Context.Files, f)
			fileLookup[f] = struct{}{}
		}
	}

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Project context updated, but failed to reload context: %v", err)}, false
	}

	var payload strings.Builder
	payload.WriteString("Project context updated.")

	summary := formatContextSummary(&cfg.Context)
	if summary != "" {
		payload.WriteString("\n")
		payload.WriteString(summary)
	}
	if len(invalidPaths) > 0 {
		payload.WriteString(fmt.Sprintf("\nWarning: The following paths do not exist and were ignored: %s", strings.Join(invalidPaths, ", ")))
	}

	return CommandOutput{Type: CommandResultString, Payload: payload.String()}, true
}
