package ui

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

type BashToolRenderer struct {
	DefaultToolRenderer
}

func (b *BashToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	cmd := ExtractToolJSONField(call.Arguments, "command")
	if cmd == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		cmd = strings.TrimSpace(call.Arguments)
	}

	prefix := ToolCallStyle.Render("⚡ bash")
	if cmd == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s $ %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(cmd, 60)))
}

func (b *BashToolRenderer) RenderExpandedCall(call types.ToolCall, viewportWidth int) string {
	cmd := ExtractToolJSONField(call.Arguments, "command")
	if cmd == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		cmd = strings.TrimSpace(call.Arguments)
	}
	prefix := ToolCallStyle.Render("⚡ bash")
	if cmd == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s $ %s", prefix, ToolMutedStyle.Render(cmd))
}

func (b *BashToolRenderer) RenderConfirmPrompt(arguments string, width int) string {
	cmd := ExtractToolJSONField(arguments, "command")
	if cmd == "" && !strings.HasPrefix(strings.TrimSpace(arguments), "{") {
		cmd = strings.TrimSpace(arguments)
	}
	return fmt.Sprintf("%s\n%s",
		ToolMutedStyle.Render("Command to execute:"),
		CommandInputStyle.Width(width).Render("$ "+cmd),
	)
}

func (b *BashToolRenderer) RenderExpandedResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" || trimmed == "(no output)" {
		return fmt.Sprintf("↳ %s", ToolMutedStyle.Render("(no output)"))
	}

	lines := strings.Split(trimmed, "\n")
	isError := strings.HasPrefix(strings.TrimSpace(lines[0]), "Error:") ||
		strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "Command exited with error:")

	indicator := ToolSuccessStyle.Render("✓")
	if isError {
		indicator = ToolErrorStyle.Render("✗")
	}

	maxLines := 30
	displayLines := lines
	if len(lines) > maxLines {
		head := lines[:15]
		tail := lines[len(lines)-10:]
		displayLines = append(append(head, fmt.Sprintf("... [%d lines omitted] ...", len(lines)-25)), tail...)
	}

	var formatted []string
	for _, l := range displayLines {
		formatted = append(formatted, "    "+ToolResultStyle.Render(l))
	}
	return fmt.Sprintf("↳ %s\n%s", indicator, strings.Join(formatted, "\n"))
}

func init() {
	DefaultToolRegistry.Register("bash", &BashToolRenderer{})
}
