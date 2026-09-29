package core

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/types"
)

type ToolViewRenderer interface {
	RenderCall(call types.ToolCall, viewportWidth int) string
	RenderResult(output string, callID string, viewportWidth int) string
	RenderExpandedCall(call types.ToolCall, viewportWidth int) string
	RenderExpandedResult(output string, callID string, viewportWidth int) string
}

type ToolRendererRegistry struct {
	mu        sync.RWMutex
	renderers map[string]ToolViewRenderer
	fallback  ToolViewRenderer
}

func NewToolRendererRegistry() *ToolRendererRegistry {
	return &ToolRendererRegistry{
		renderers: make(map[string]ToolViewRenderer),
		fallback:  &DefaultToolRenderer{},
	}
}

func (r *ToolRendererRegistry) Register(name string, renderer ToolViewRenderer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.renderers[name] = renderer
}

func (r *ToolRendererRegistry) Get(name string) ToolViewRenderer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if renderer, ok := r.renderers[name]; ok {
		return renderer
	}
	return r.fallback
}

var DefaultToolRegistry = NewToolRendererRegistry()

type DefaultToolRenderer struct{}

func (d *DefaultToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	summary := SummarizeToolArgs(call.Arguments)
	prefix := ToolCallStyle.Render("⚡ " + call.Name)
	if summary == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s(%s)", prefix, ToolMutedStyle.Render(summary))
}

func (d *DefaultToolRenderer) RenderExpandedCall(call types.ToolCall, viewportWidth int) string {
	return d.RenderCall(call, viewportWidth)
}

func (d *DefaultToolRenderer) RenderExpandedResult(output string, callID string, viewportWidth int) string {
	return d.RenderResult(output, callID, viewportWidth)
}

func (d *DefaultToolRenderer) RenderResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" || trimmed == "(no output)" {
		return fmt.Sprintf("↳ %s", ToolMutedStyle.Render("(no output)"))
	}

	lines := strings.Split(trimmed, "\n")
	firstLine := strings.TrimSpace(lines[0])
	lastLine := strings.TrimSpace(lines[len(lines)-1])

	isError := strings.HasPrefix(firstLine, "Error:") ||
		strings.HasPrefix(firstLine, "error:") ||
		strings.HasPrefix(lastLine, "Command exited with error:")

	if isError {
		errorSummary := firstLine
		if strings.HasPrefix(lastLine, "Command exited with error:") {
			errorSummary = lastLine
		}
		clean := strings.TrimPrefix(errorSummary, "Error: ")
		clean = strings.TrimPrefix(clean, "error: ")
		summary := TruncateSingleLine(clean, 80)
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(summary))
	}
	summary := TruncateSingleLine(firstLine, 80)
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(summary))
}

func ExtractToolJSONField(arguments string, field string) string {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return ""
	}

	var rawString string
	if err := json.Unmarshal([]byte(trimmed), &rawString); err == nil && strings.HasPrefix(strings.TrimSpace(rawString), "{") {
		trimmed = strings.TrimSpace(rawString)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		return ""
	}

	val, ok := data[field]
	if !ok {
		return ""
	}
	if nestedMap, ok := val.(map[string]any); ok {
		if b, err := json.Marshal(nestedMap); err == nil {
			return string(b)
		}
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}
