package core

import (
	"fmt"

	"github.com/sokinpui/coder/internal/types"
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

func init() {
	DefaultToolRegistry.Register("write", &WriteToolRenderer{})
}
