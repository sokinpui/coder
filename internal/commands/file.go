package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	registerCommand("file", fileCmd, "add path to context", PathArgumentCompleter)
}

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

		results = append(results, fullPath)
	}
	return results
}

func fileCmd(args string, s SessionController) (CommandOutput, bool) {
	paths := strings.Fields(args)

	if len(paths) == 0 {
		s.SetContextFiles([]string{})
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

	currentFiles := s.GetContextFiles()
	// Combine dirs and specific files, then filter with default exclusions
	newResolvedFiles, _ := utils.SourceToFileList(dirs, files, source.Exclusions)

	s.SetContextFiles(AppendUnique(currentFiles, newResolvedFiles))

	if err := s.LoadContext(); err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Project context updated, but failed to reload context: %v", err)}, false
	}

	var payload strings.Builder
	payload.WriteString("Project context updated.")

	summary := formatFileListSummary(s.GetContextFiles())
	if summary != "" {
		payload.WriteString("\n")
		payload.WriteString(summary)
	}
	if len(invalidPaths) > 0 {
		fmt.Fprintf(&payload, "\nWarning: The following paths do not exist and were ignored: %s", strings.Join(invalidPaths, ", "))
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: payload.String()}, true
}
