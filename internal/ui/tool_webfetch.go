package ui

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

type WebFetchToolRenderer struct {
	DefaultToolRenderer
}

func (w *WebFetchToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	url := ExtractToolJSONField(call.Arguments, "url")
	startLine := ExtractToolJSONField(call.Arguments, "start_line")
	endLine := ExtractToolJSONField(call.Arguments, "end_line")
	prefix := ToolCallStyle.Render("⚡ webfetch")
	if url == "" {
		return prefix + "()"
	}
	lineRange := formatLineRange(startLine, endLine)
	target := TruncateSingleLine(url, 60) + lineRange
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(target))
}

func (w *WebFetchToolRenderer) RenderDetailedCall(call types.ToolCall, mode ToolViewMode, viewportWidth int) string {
	if mode == ToolViewCompact {
		return w.RenderCall(call, viewportWidth)
	}
	url := ExtractToolJSONField(call.Arguments, "url")
	prompt := ExtractToolJSONField(call.Arguments, "prompt")
	startLine := ExtractToolJSONField(call.Arguments, "start_line")
	endLine := ExtractToolJSONField(call.Arguments, "end_line")
	prefix := ToolCallStyle.Render("⚡ webfetch")
	if url == "" {
		return prefix + "()"
	}
	lineRange := formatLineRange(startLine, endLine)
	target := url + lineRange
	if prompt == "" {
		return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(target))
	}
	return fmt.Sprintf("%s %s\n    %s", prefix, ToolMutedStyle.Render(target), ToolMutedStyle.Render("Prompt: "+prompt))
}

func formatLineRange(startLine, endLine string) string {
	startLine = strings.TrimSpace(startLine)
	endLine = strings.TrimSpace(endLine)

	if startLine == "" && endLine == "" {
		return ""
	}
	if startLine != "" && endLine != "" {
		return fmt.Sprintf(" [L%s-L%s]", startLine, endLine)
	}
	if startLine != "" {
		return fmt.Sprintf(" [L%s+]", startLine)
	}
	return fmt.Sprintf(" [1-L%s]", endLine)
}

func (w *WebFetchToolRenderer) RenderConfirmPrompt(arguments string, width int) string {
	url := ExtractToolJSONField(arguments, "url")
	prompt := ExtractToolJSONField(arguments, "prompt")
	return fmt.Sprintf("%s %s\n%s %s",
		ToolMutedStyle.Render("URL:"),
		url,
		ToolMutedStyle.Render("Prompt:"),
		TruncateSingleLine(prompt, width),
	)
}

func (w *WebFetchToolRenderer) RenderResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "failed to") {
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(TruncateSingleLine(trimmed, 80)))
	}
	lines := strings.Split(trimmed, "\n")
	firstLine := strings.TrimSpace(lines[0])
	summary := TruncateSingleLine(firstLine, 80)
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(summary))
}

func (w *WebFetchToolRenderer) RenderDetailedResult(output string, callID string, mode ToolViewMode, viewportWidth int) string {
	if mode == ToolViewCompact {
		return w.RenderResult(output, callID, viewportWidth)
	}
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "failed to") {
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
	DefaultToolRegistry.Register("webfetch", &WebFetchToolRenderer{})
}
