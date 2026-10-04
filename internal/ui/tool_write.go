package ui

import (
	"fmt"
	"path/filepath"
	"strings"

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

func (w *WriteToolRenderer) RenderExpandedCall(call types.ToolCall, viewportWidth int) string {
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

	lines := strings.Split(content, "\n")
	previewContent := content
	if len(lines) > 40 {
		head := strings.Join(lines[:20], "\n")
		tail := strings.Join(lines[len(lines)-10:], "\n")
		previewContent = fmt.Sprintf("%s\n... [%d lines omitted] ...\n%s", head, len(lines)-30, tail)
	}

	codeBlock := fmt.Sprintf("```%s\n%s\n```", lang, previewContent)
	renderedCode := markdown.Render(codeBlock, viewportWidth)
	return fmt.Sprintf("%s\n%s", header, renderedCode)
}

func (w *WriteToolRenderer) RenderExpandedResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "Error:") {
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(trimmed))
	}
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(trimmed))
}

func init() {
	DefaultToolRegistry.Register("write", &WriteToolRenderer{})
}
