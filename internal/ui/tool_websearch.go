package ui

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

type WebSearchToolRenderer struct {
	DefaultToolRenderer
}

func (w *WebSearchToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	query := ExtractToolJSONField(call.Arguments, "query")
	if query == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		query = strings.TrimSpace(call.Arguments)
	}
	prefix := ToolCallStyle.Render("⚡ websearch")
	if query == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(query, 60)))
}

func (w *WebSearchToolRenderer) RenderDetailedCall(call types.ToolCall, mode ToolViewMode, viewportWidth int) string {
	if mode == ToolViewCompact {
		return w.RenderCall(call, viewportWidth)
	}
	query := ExtractToolJSONField(call.Arguments, "query")
	if query == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		query = strings.TrimSpace(call.Arguments)
	}
	prefix := ToolCallStyle.Render("⚡ websearch")
	if query == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(query))
}

func (w *WebSearchToolRenderer) RenderConfirmPrompt(arguments string, width int) string {
	query := ExtractToolJSONField(arguments, "query")
	if query == "" && !strings.HasPrefix(strings.TrimSpace(arguments), "{") {
		query = strings.TrimSpace(arguments)
	}
	return fmt.Sprintf("%s %s",
		ToolMutedStyle.Render("Search query:"),
		TruncateSingleLine(query, width),
	)
}

func (w *WebSearchToolRenderer) RenderResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "search failed:") {
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(TruncateSingleLine(trimmed, 80)))
	}
	if trimmed == "No results found." {
		return fmt.Sprintf("↳ %s %s", ToolMutedStyle.Render("✓"), ToolResultStyle.Render("No results found"))
	}
	lines := strings.Split(trimmed, "\n")
	firstLine := strings.TrimSpace(lines[0])
	summary := TruncateSingleLine(firstLine, 80)
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(summary))
}

func (w *WebSearchToolRenderer) RenderDetailedResult(output string, callID string, mode ToolViewMode, viewportWidth int) string {
	if mode == ToolViewCompact {
		return w.RenderResult(output, callID, viewportWidth)
	}
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "search failed:") {
		return fmt.Sprintf("↳ %s\n%s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(trimmed))
	}

	lines := strings.Split(trimmed, "\n")
	displayLines := lines
	if mode == ToolViewSummary && len(lines) > 16 {
		head := lines[:8]
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
	return fmt.Sprintf("↳ %s\n%s", ToolSuccessStyle.Render("✓"), strings.Join(formatted, "\n"))
}

func init() {
	DefaultToolRegistry.Register("websearch", &WebSearchToolRenderer{})
}
