package core

import (
	"encoding/json"
	"strings"

	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/core/markdown"
)

func RenderMessage(msg types.Message, viewportWidth int, renderer *markdown.Renderer) string {
	content := msg.Content
	switch msg.Type {
	case types.InitMessage:
		return InitMessageStyle.Width(viewportWidth - InitMessageStyle.GetHorizontalFrameSize()).Render(content)
	case types.DirectoryMessage:
		return DirectoryWelcomeStyle.Width(viewportWidth - DirectoryWelcomeStyle.GetHorizontalFrameSize()).Render(content)
	case types.UserMessage:
		return UserInputStyle.Width(viewportWidth - UserInputStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandMessage, types.ShellCmdMessage, types.ContextCmdMessage,
		types.FileApplyCmdMessage, types.FileApplyUndoCmdMessage:
		prefix := ""
		if msg.Type == types.ShellCmdMessage {
			prefix = "Shell: "
		}
		return CommandInputStyle.Width(viewportWidth - CommandInputStyle.GetHorizontalFrameSize()).Render(prefix + content)
	case types.ImageMessage:
		if msg.IsDocumentImage() {
			return ""
		}
		return ImageMessageStyle.Width(viewportWidth - ImageMessageStyle.GetHorizontalFrameSize()).Render("Image: " + content)
	case types.AIMessage:
		var parts []string
		if len(msg.ToolCalls) > 0 {
			tcPart := RenderToolCallMessage(msg, false, viewportWidth)
			if tcPart != "" {
				parts = append(parts, tcPart)
			}
		}
		if content != "" {
			if renderer != nil {
				renderedAI, err := renderer.Render(content)
				if err == nil {
					parts = append(parts, renderedAI)
				} else {
					parts = append(parts, content)
				}
			} else {
				parts = append(parts, content)
			}
		}
		return strings.Join(parts, "\n")
	case types.ToolCallMessage:
		return RenderToolCallMessage(msg, false, viewportWidth)
	case types.ToolResultMessage:
		return RenderToolResultMessage(msg, "", false, viewportWidth)
	case types.CommandResultMessage, types.ShellCmdResultMessage, types.ContextCmdResultMessage,
		types.FileApplyCmdResultMessage, types.FileApplyUndoCmdResultMessage:
		return CommandResultStyle.Width(viewportWidth - CommandResultStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandErrorResultMessage,
		types.FileApplyCmdErrorMessage, types.FileApplyUndoCmdErrorMessage:
		return CommandErrorStyle.Width(viewportWidth - CommandErrorStyle.GetHorizontalFrameSize()).Render(content)
	default:
		return ""
	}
}

func RenderToolCallMessage(msg types.Message, expanded bool, viewportWidth ...int) string {
	width := 80
	if len(viewportWidth) > 0 && viewportWidth[0] > 0 {
		width = viewportWidth[0]
	}

	if len(msg.ToolCalls) == 0 {
		if msg.Content == "" {
			return ""
		}
		return ToolCallStyle.Render("⚡ " + msg.Content)
	}

	var lines []string
	for _, tc := range msg.ToolCalls {
		renderer := DefaultToolRegistry.Get(tc.Name)
		if expanded {
			lines = append(lines, renderer.RenderExpandedCall(tc, width))
		} else {
			lines = append(lines, renderer.RenderCall(tc, width))
		}
	}
	return strings.Join(lines, "\n")
}

func RenderToolResultMessage(msg types.Message, toolName string, expanded bool, viewportWidth ...int) string {
	width := 80
	if len(viewportWidth) > 0 && viewportWidth[0] > 0 {
		width = viewportWidth[0]
	}
	renderer := DefaultToolRegistry.fallback
	if toolName != "" {
		renderer = DefaultToolRegistry.Get(toolName)
	}
	if expanded {
		return renderer.RenderExpandedResult(msg.Content, msg.ToolCallID, width)
	}
	return renderer.RenderResult(msg.Content, msg.ToolCallID, width)
}

func SummarizeToolArgs(args string) string {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return ""
	}

	var rawString string
	if err := json.Unmarshal([]byte(trimmed), &rawString); err == nil && strings.HasPrefix(strings.TrimSpace(rawString), "{") {
		trimmed = strings.TrimSpace(rawString)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(trimmed), &m); err == nil {
		for _, key := range []string{"path", "command", "query"} {
			if val, ok := m[key]; ok {
				if s, ok := val.(string); ok && s != "" {
					return TruncateSingleLine(s, 60)
				}
			}
		}
	}
	return TruncateSingleLine(trimmed, 60)
}

func TruncateSingleLine(s string, maxLen int) string {
	clean := strings.Join(strings.Fields(s), " ")
	if len(clean) <= maxLen {
		return clean
	}
	return clean[:maxLen-3] + "..."
}
