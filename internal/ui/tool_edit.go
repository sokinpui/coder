package ui

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

type EditToolRenderer struct {
	DefaultToolRenderer
}

func (e *EditToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	prefix := ToolCallStyle.Render("⚡ edit")
	if path == "" {
		return e.DefaultToolRenderer.RenderCall(call, viewportWidth)
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(path, 60)))
}

func (e *EditToolRenderer) RenderExpandedCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	oldStr := ExtractToolJSONField(call.Arguments, "old_string")
	newStr := ExtractToolJSONField(call.Arguments, "new_string")

	header := fmt.Sprintf("%s %s", ToolCallStyle.Render("⚡ edit"), ToolMutedStyle.Render(path))
	if oldStr == "" && newStr == "" {
		return header
	}

	var diffBuilder strings.Builder
	diffBuilder.WriteString("```diff\n")
	if path != "" {
		diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ b/%s\n", path, path))
	}
	for line := range strings.SplitSeq(oldStr, "\n") {
		diffBuilder.WriteString("-" + line + "\n")
	}
	for line := range strings.SplitSeq(newStr, "\n") {
		diffBuilder.WriteString("+" + line + "\n")
	}
	diffBuilder.WriteString("```")

	renderedDiff := markdown.Render(diffBuilder.String(), viewportWidth)
	return fmt.Sprintf("%s\n%s", header, renderedDiff)
}

func (e *EditToolRenderer) RenderExpandedResult(output string, callID string, viewportWidth int) string {
	if strings.HasPrefix(strings.TrimSpace(output), "Error:") {
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(strings.TrimSpace(output)))
	}
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(strings.TrimSpace(output)))
}

func init() {
	DefaultToolRegistry.Register("edit", &EditToolRenderer{})
}
