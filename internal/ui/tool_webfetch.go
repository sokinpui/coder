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
	prefix := ToolCallStyle.Render("⚡ webfetch")
	if url == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(url, 60)))
}

func (w *WebFetchToolRenderer) RenderExpandedCall(call types.ToolCall, viewportWidth int) string {
	url := ExtractToolJSONField(call.Arguments, "url")
	prompt := ExtractToolJSONField(call.Arguments, "prompt")
	prefix := ToolCallStyle.Render("⚡ webfetch")
	if url == "" {
		return prefix + "()"
	}
	if prompt == "" {
		return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(url))
	}
	return fmt.Sprintf("%s %s\n    %s", prefix, ToolMutedStyle.Render(url), ToolMutedStyle.Render("Prompt: "+prompt))
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

func (w *WebFetchToolRenderer) RenderExpandedResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") || strings.HasPrefix(trimmed, "failed to") {
		return fmt.Sprintf("↳ %s\n%s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(trimmed))
	}

	lines := strings.Split(trimmed, "\n")
	maxLines := 25
	displayLines := lines
	if len(lines) > maxLines {
		displayLines = append(lines[:maxLines], fmt.Sprintf("... [%d lines omitted]", len(lines)-maxLines))
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
