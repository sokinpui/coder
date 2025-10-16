package commands

import (
	"fmt"
	"os"
	"strings"
)

func init() {
	registerCommand("file", fileCmd, nil)
}

func fileCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)
	cfg := s.GetConfig()

	if len(paths) == 0 {
		cfg.Sources.Dirs = []string{}
		cfg.Sources.Files = []string{}
		if err := s.LoadContext(); err != nil {
			msg := fmt.Sprintf("Project source files cleared, but failed to reload context: %v", err)
			return CommandOutput{Type: CommandResultString, Payload: msg}, false
		}
		return CommandOutput{Type: CommandResultString, Payload: "Project source files cleared. The next prompt will not include any project source code."}, true
	}

	var files []string
	var dirs []string
	var invalidPaths []string

	for _, p := range paths {
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
	for _, d := range cfg.Sources.Dirs {
		dirLookup[d] = struct{}{}
	}
	for _, d := range dirs {
		if _, exists := dirLookup[d]; !exists {
			cfg.Sources.Dirs = append(cfg.Sources.Dirs, d)
			dirLookup[d] = struct{}{}
		}
	}

	// De-duplicate and append files
	fileLookup := make(map[string]struct{})
	for _, f := range cfg.Sources.Files {
		fileLookup[f] = struct{}{}
	}
	for _, f := range files {
		if _, exists := fileLookup[f]; !exists {
			cfg.Sources.Files = append(cfg.Sources.Files, f)
			fileLookup[f] = struct{}{}
		}
	}

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Project source updated, but failed to reload context: %v", err)}, false
	}

	var payload strings.Builder
	fmt.Fprintln(&payload, "Project source updated.")
	if len(cfg.Sources.Dirs) > 0 {
		fmt.Fprintf(&payload, "Directories: %s\n", strings.Join(cfg.Sources.Dirs, ", "))
	}
	if len(cfg.Sources.Files) > 0 {
		fmt.Fprintf(&payload, "Files: %s\n", strings.Join(cfg.Sources.Files, ", "))
	}
	if len(invalidPaths) > 0 {
		fmt.Fprintf(&payload, "Warning: The following paths do not exist and were ignored: %s\n", strings.Join(invalidPaths, ", "))
	}

	return CommandOutput{Type: CommandResultString, Payload: strings.TrimSpace(payload.String())}, true
}
