package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	registerCommand("file", fileCmd, PathArgumentCompleter)
}

// PathArgumentCompleter provides file and directory path completions based on the prefix.
func PathArgumentCompleter(cfg *config.Config, prefix string) []string {
	dir := "."
	if lastSlash := strings.LastIndex(prefix, "/"); lastSlash != -1 {
		dir = prefix[:lastSlash+1]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []string
	for _, entry := range entries {
		name := entry.Name()
		fullPath := name
		if dir != "." {
			fullPath = filepath.Join(dir, name)
		}

		fullPath = filepath.ToSlash(fullPath)
		if entry.IsDir() {
			fullPath += "/"
		}

		if fullPath == prefix {
			continue
		}
		results = append(results, fullPath)
	}
	return results
}

func fileCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)
	cfg := s.GetConfig()

	if len(paths) == 0 {
		cfg.Context.Dirs = []string{}
		cfg.Context.Files = []string{}
		if err := s.LoadContext(); err != nil {
			msg := fmt.Sprintf("Project context cleared, but failed to reload context: %v", err)
			return CommandOutput{Type: types.MessagesUpdated, Payload: msg}, false
		}
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Project context cleared. The next prompt will not include any project source code."}, true
	}

	var files []string
	var dirs []string
	var invalidPaths []string

	expandedPaths, invalidPatterns := ExpandPaths(paths)
	invalidPaths = append(invalidPaths, invalidPatterns...)

	for _, p := range expandedPaths {
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				invalidPaths = append(invalidPaths, p)
				continue
			}
			return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error accessing path %s: %v", p, err)}, false
		}
		if info.IsDir() {
			dirs = append(dirs, p)
		} else {
			files = append(files, p)
		}
	}

	cfg.Context.Dirs = AppendUnique(cfg.Context.Dirs, dirs)
	cfg.Context.Files = AppendUnique(cfg.Context.Files, files)

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project context updated, but failed to reload context: %v", err)}, false
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

	return CommandOutput{Type: types.MessagesUpdated, Payload: payload.String()}, true
}
