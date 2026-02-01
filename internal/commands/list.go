package commands

import (
	"github.com/sokinpui/coder/internal/source"
	"github.com/sokinpui/coder/internal/utils"
	"github.com/sokinpui/pcat"
	"fmt"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("list", listCmd, nil)
	registerCommand("list-all", listFullCmd, nil)
}

func listFullCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	context := &cfg.Context

	if len(context.Dirs) == 0 && len(context.Files) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No project source files or directories are set."}, true
	}

	finalExclusions := append(source.Exclusions, context.Exclusions...)
	allFiles, err := utils.SourceToFileList(context.Dirs, context.Files, finalExclusions)
	if err != nil {
		return CommandOutput{Type: types.MessagesUpdated, Payload: fmt.Sprintf("Error listing source files: %v", err)}, false
	}

	if len(allFiles) == 0 {
		overview := formatContextSummary(context)
		summary := "Current project context:\n" + overview + "\n\n" + "No files found in the current context."
		return CommandOutput{Type: types.MessagesUpdated, Payload: summary}, true
	}

	fileList, err := pcat.Run(
		allFiles, // specificFiles
		nil,      // directories
		nil,      // extensions
		nil,      // excludePatterns
		false,    // withLineNumbers
		true,     // hidden
		true,     // listOnly
	)

	overview := formatContextSummary(context)

	summary := "Current project context:\n" + overview + "\n\n" + "Files read by AI:\n" + fileList

	payload := summary

	return CommandOutput{Type: types.MessagesUpdated, Payload: payload}, true
}

func listCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	context := &cfg.Context

	if len(context.Dirs) == 0 && len(context.Files) == 0 {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No project source files or directories are set."}, true
	}

	overview := formatContextSummary(context)

	summary := "Current project context:\n" + overview

	payload := summary

	return CommandOutput{Type: types.MessagesUpdated, Payload: payload}, true
}
