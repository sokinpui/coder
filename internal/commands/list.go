package commands

import (
	"coder/internal/source"
	"coder/internal/utils"
	"fmt"
	"os/exec"
)

func init() {
	registerCommand("list", listCmd, nil)
}

func listCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	sources := &cfg.Sources

	if len(sources.Dirs) == 0 && len(sources.Files) == 0 {
		return CommandOutput{Type: CommandResultString, Payload: "No project source files or directories are set."}, true
	}

	allFiles, err := utils.SourceToFileList(sources.Dirs, sources.Files, source.Exclusions)
	if err != nil {
		return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error listing source files: %v", err)}, false
	}

	pcatArgs := append([]string{"--no-header", "-l"}, allFiles...)
	cmd := exec.Command("pcat", pcatArgs...)
	output, err := cmd.CombinedOutput()

	payload := "Current project sources:\n" + string(output)

	return CommandOutput{Type: CommandResultString, Payload: payload}, true
}
