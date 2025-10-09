package commands

import (
	"coder/internal/config"
	"coder/internal/core"
	"fmt"
	"os"
	"strings"
)

func init() {
	registerCommand("file", fileCmd, nil)
}

func fileCmd(args string, messages []core.Message, cfg *config.Config, sess SessionChanger) (CommandOutput, bool) {
	paths := strings.Fields(args)

	if len(paths) == 0 {
		cfg.Sources.FileDirs = []string{}
		cfg.Sources.FilePaths = []string{}
		return CommandOutput{Type: CommandResultString, Payload: "Project source files cleared. The next prompt will not include any project source code."}, true
	}

	var filePaths []string
	var dirPaths []string
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
			dirPaths = append(dirPaths, p)
		} else {
			filePaths = append(filePaths, p)
		}
	}

	// De-duplicate and append directories
	dirLookup := make(map[string]struct{})
	for _, d := range cfg.Sources.FileDirs {
		dirLookup[d] = struct{}{}
	}
	for _, d := range dirPaths {
		if _, exists := dirLookup[d]; !exists {
			cfg.Sources.FileDirs = append(cfg.Sources.FileDirs, d)
			dirLookup[d] = struct{}{}
		}
	}

	// De-duplicate and append files
	fileLookup := make(map[string]struct{})
	for _, f := range cfg.Sources.FilePaths {
		fileLookup[f] = struct{}{}
	}
	for _, f := range filePaths {
		if _, exists := fileLookup[f]; !exists {
			cfg.Sources.FilePaths = append(cfg.Sources.FilePaths, f)
			fileLookup[f] = struct{}{}
		}
	}

	var payload strings.Builder
	fmt.Fprintln(&payload, "Project source updated.")
	if len(cfg.Sources.FileDirs) > 0 {
		fmt.Fprintf(&payload, "Directories: %s\n", strings.Join(cfg.Sources.FileDirs, ", "))
	}
	if len(cfg.Sources.FilePaths) > 0 {
		fmt.Fprintf(&payload, "Files: %s\n", strings.Join(cfg.Sources.FilePaths, ", "))
	}
	if len(invalidPaths) > 0 {
		fmt.Fprintf(&payload, "Warning: The following paths do not exist and were ignored: %s\n", strings.Join(invalidPaths, ", "))
	}

	return CommandOutput{Type: CommandResultString, Payload: strings.TrimSpace(payload.String())}, true
}
