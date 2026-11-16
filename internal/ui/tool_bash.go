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
	return fmt.Sprintf("%s %s", prefix, ToolCallArgStyle.Render("$ "+TruncateSingleLine(cmd, 60)))
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
	return fmt.Sprintf("%s %s", prefix, ToolCallArgStyle.Render("$ "+cmd))
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

	var formatted []string
	for _, l := range lines {
		formatted = append(formatted, "    "+ToolResultStyle.Render(l))
	}
	return fmt.Sprintf("↳ %s\n%s", indicator, strings.Join(formatted, "\n"))
}

func init() {
	DefaultToolRegistry.Register("bash", &BashToolRenderer{})
}
