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
	fileList, err := cmd.CombinedOutput()

	overview := formatSourceSummary(sources)

	summary := "Current project sources:\n" + overview + "\n\n" + "Files read by AI\n:" + string(fileList)

	payload := summary

	return CommandOutput{Type: CommandResultString, Payload: payload}, true
}
