package commands

import (
	"fmt"
	"strings"
)

func init() {
	registerCommand("list", listCmd, nil)
}

func listCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	sources := cfg.Sources

	if len(sources.Dirs) == 0 && len(sources.Files) == 0 {
		return CommandOutput{Type: CommandResultString, Payload: "No project source files or directories are set."}, true
	}

	var b strings.Builder
	fmt.Fprintln(&b, "Current project sources:")
	if len(sources.Dirs) > 0 {
		fmt.Fprintf(&b, "Directories: %s\n", strings.Join(sources.Dirs, ", "))
	}
	if len(sources.Files) > 0 {
		fmt.Fprintf(&b, "Files: %s\n", strings.Join(sources.Files, ", "))
	}

	return CommandOutput{Type: CommandResultString, Payload: strings.TrimSpace(b.String())}, true
}
