package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

type WriteToolRenderer struct {
	DefaultToolRenderer
}

func (w *WriteToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	prefix := ToolCallStyle.Render("⚡ write")
	if path == "" {
		return w.DefaultToolRenderer.RenderCall(call, viewportWidth)
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(path, 60)))
}

func (w *WriteToolRenderer) RenderDetailedCall(call types.ToolCall, mode ToolViewMode, viewportWidth int) string {
	if mode == ToolViewCompact {
		return w.RenderCall(call, viewportWidth)
	}
	path := ExtractToolJSONField(call.Arguments, "path")
	content := ExtractToolJSONField(call.Arguments, "content")

	header := fmt.Sprintf("%s %s", ToolCallStyle.Render("⚡ write"), ToolMutedStyle.Render(path))
	if content == "" {
		return header
	}

	lang := strings.TrimPrefix(filepath.Ext(path), ".")
	if lang == "" {
		lang = "txt"
	}

	previewContent := content
	if mode == ToolViewSummary {
		lines := strings.Split(content, "\n")
		if len(lines) > 24 {
			head := strings.Join(lines[:12], "\n")
			tail := strings.Join(lines[len(lines)-8:], "\n")
			previewContent = fmt.Sprintf("%s\n... [%d lines omitted] ...\n%s", head, len(lines)-20, tail)
		}
	}

	cardWidth := max(30, viewportWidth-4)
	contentWidth := max(20, cardWidth-PreviewBoxStyle.GetHorizontalFrameSize())
	codeBlock := fmt.Sprintf("```%s\n%s\n```", lang, previewContent)
	rendered := markdown.Render(codeBlock, contentWidth)
	lines := strings.Split(strings.ReplaceAll(rendered, "\r\n", "\n"), "\n")
	lines = trimVerticalBlankLines(lines)
	renderedCode := strings.Join(lines, "\n")
	styledBox := PreviewBoxStyle.Width(cardWidth).Render(renderedCode)
	return fmt.Sprintf("%s\n%s", header, styledBox)
}

func trimVerticalBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[0])) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func (w *WriteToolRenderer) RenderConfirmPrompt(arguments string, width int) string {
	path := ExtractToolJSONField(arguments, "path")
	content := ExtractToolJSONField(arguments, "content")

	header := fmt.Sprintf("%s (%d bytes)", ToolMutedStyle.Render("Write to file: ")+path, len(content))
	if len(content) == 0 {
		return header
	}

	lines := strings.Split(content, "\n")
	preview := content
	if len(lines) > 10 {
		preview = strings.Join(lines[:10], "\n") + "\n... [truncated]"
	}
	return fmt.Sprintf("%s\n\n%s", header, ToolResultStyle.Render(preview))
}

func (w *WriteToolRenderer) RenderDetailedResult(output string, callID string, mode ToolViewMode, viewportWidth int) string {
	if mode == ToolViewCompact {
		return w.RenderResult(output, callID, viewportWidth)
	}
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") {
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(trimmed))
	}
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(trimmed))
}

func init() {
	DefaultToolRegistry.Register("write", &WriteToolRenderer{})
}
