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

func (b *BashToolRenderer) RenderDetailedCall(call types.ToolCall, mode ToolViewMode, viewportWidth int) string {
	cmd := ExtractToolJSONField(call.Arguments, "command")
	if cmd == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		cmd = strings.TrimSpace(call.Arguments)
	}
	if mode == ToolViewCompact {
		return b.RenderCall(call, viewportWidth)
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

func (b *BashToolRenderer) RenderDetailedResult(output string, callID string, mode ToolViewMode, viewportWidth int) string {
	if mode == ToolViewCompact {
		return b.RenderResult(output, callID, viewportWidth)
	}

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

	displayLines := lines
	if mode == ToolViewSummary && len(lines) > 14 {
		head := lines[:7]
		tail := lines[len(lines)-5:]
		displayLines = make([]string, 0, len(head)+len(tail)+1)
		displayLines = append(displayLines, head...)
		displayLines = append(displayLines, fmt.Sprintf("... [%d lines omitted] ...", len(lines)-(len(head)+len(tail))))
		displayLines = append(displayLines, tail...)
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
