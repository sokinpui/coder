package core

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

type ReadToolRenderer struct {
	DefaultToolRenderer
}

func (r *ReadToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	path := ExtractToolJSONField(call.Arguments, "path")
	if path == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		path = strings.TrimSpace(call.Arguments)
	}

	prefix := ToolCallStyle.Render("⚡ read")
	if path == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(path, 60)))
}

func init() {
	DefaultToolRegistry.Register("read", &ReadToolRenderer{})
}
