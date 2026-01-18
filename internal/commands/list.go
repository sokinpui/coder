package commands

import (
	"coder/internal/source"
	"coder/internal/utils"
	"fmt"
	"os/exec"
)

func init() {
	registerCommand("list", listCmd, nil)
	registerCommand("list-all", listFullCmd, nil)
}

func listFullCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	context := &cfg.Context

	if len(context.Dirs) == 0 && len(context.Files) == 0 {
		return CommandOutput{Type: CommandResultString, Payload: "No project source files or directories are set."}, true
	}

	finalExclusions := append(source.Exclusions, context.Exclusions...)
	allFiles, err := utils.SourceToFileList(context.Dirs, context.Files, finalExclusions)
	if err != nil {
		return CommandOutput{Type: CommandResultString, Payload: fmt.Sprintf("Error listing source files: %v", err)}, false
	}

	if len(allFiles) == 0 {
		overview := formatContextSummary(context)
		summary := "Current project context:\n" + overview + "\n\n" + "No files found in the current context."
		return CommandOutput{Type: CommandResultString, Payload: summary}, true
	}

	pcatArgs := append([]string{"-l"}, allFiles...)
	cmd := exec.Command("pcat", pcatArgs...)
	fileList, err := cmd.CombinedOutput()

	overview := formatContextSummary(context)

	summary := "Current project context:\n" + overview + "\n\n" + "Files read by AI:\n" + string(fileList)

	payload := summary

	return CommandOutput{Type: CommandResultString, Payload: payload}, true
}

func listCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	context := &cfg.Context

	if len(context.Dirs) == 0 && len(context.Files) == 0 {
		return CommandOutput{Type: CommandResultString, Payload: "No project source files or directories are set."}, true
	}

	overview := formatContextSummary(context)

	summary := "Current project context:\n" + overview

	payload := summary

	return CommandOutput{Type: CommandResultString, Payload: payload}, true
}
