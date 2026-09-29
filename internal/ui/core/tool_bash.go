package core

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

type BashToolRenderer struct {
	DefaultToolRenderer
}

func (b *BashToolRenderer) RenderCall(call types.ToolCall, viewportWidth int) string {
	cmd := ExtractToolJSONField(call.Arguments, "command")
	if cmd == "" && !strings.HasPrefix(strings.TrimSpace(call.Arguments), "{") {
		cmd = strings.TrimSpace(call.Arguments)
	}

	prefix := ToolCallStyle.Render("⚡ bash")
	if cmd == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s $ %s", prefix, ToolMutedStyle.Render(TruncateSingleLine(cmd, 60)))
}

func init() {
	DefaultToolRegistry.Register("bash", &BashToolRenderer{})
}
