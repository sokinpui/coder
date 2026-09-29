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

func (d *DefaultToolRenderer) RenderResult(output string, callID string, viewportWidth int) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" || trimmed == "(no output)" {
		return fmt.Sprintf("↳ %s", ToolMutedStyle.Render("(no output)"))
	}

	firstLine := strings.Split(trimmed, "\n")[0]
	summary := TruncateSingleLine(firstLine, 80)
	if strings.HasPrefix(trimmed, "Error:") || strings.Contains(trimmed, "Command exited with error:") {
		clean := strings.TrimPrefix(summary, "Error: ")
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(clean))
	}
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(summary))
}

func ExtractToolJSONField(arguments string, field string) string {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return ""
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		return ""
	}
	val, ok := data[field]
	if !ok {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}
