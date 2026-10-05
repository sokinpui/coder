package ui

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

type ReadToolRenderer struct {
	DefaultToolRenderer
}

func (r *ReadToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	if path == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		path = strings.TrimSpace(call.Arguments)
	}

	prefix := ToolCallStyle.Render("⚡ read")
	if path == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(path, 60)))
}

func (r *ReadToolRenderer) RenderExpandedCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	if path == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		path = strings.TrimSpace(call.Arguments)
	}

	prefix := ToolCallStyle.Render("⚡ read")
	if path == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(path))
}

func (r *ReadToolRenderer) RenderConfirmPrompt(arguments string, width int) string {
	path := ExtractToolJSONField(arguments, "path")
	if path == "" && !strings.HasPrefix(strings.TrimSpace(arguments), "{") {
		path = strings.TrimSpace(arguments)
	}
	return ToolMutedStyle.Render("Read file: ") + path
}

func (r *ReadToolRenderer) RenderResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "cannot read") {
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(trimmed))
	}
	if strings.HasPrefix(trimmed, "Successfully rendered PDF") || strings.HasPrefix(trimmed, "Successfully read image") {
		return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(trimmed))
	}
	if trimmed == "" || trimmed == "(empty file)" {
		return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render("0 lines read"))
	}

	lines := strings.Split(strings.TrimRight(output, "\r\n"), "\n")
	lineCount := len(lines)
	msg := fmt.Sprintf("%d lines read", lineCount)
	if lineCount == 1 {
		msg = "1 line read"
	}
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(msg))
}

func (r *ReadToolRenderer) RenderExpandedResult(output string, callID string, viewportWidth int) string {
	return r.RenderResult(output, callID, viewportWidth)
}

func init() {
	DefaultToolRegistry.Register("read", &ReadToolRenderer{})
}
