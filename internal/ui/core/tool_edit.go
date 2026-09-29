package core

import (
	"fmt"

	"github.com/sokinpui/coder/internal/types"
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

func init() {
	DefaultToolRegistry.Register("edit", &EditToolRenderer{})
}
