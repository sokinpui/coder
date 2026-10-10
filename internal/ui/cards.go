package ui

import (
	"encoding/json"
	"strings"

	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
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
			tcPart := RenderToolCallMessage(msg, ToolViewCompact, viewportWidth)
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
		return RenderToolCallMessage(msg, ToolViewCompact, viewportWidth)
	case types.ToolResultMessage:
		return RenderToolResultMessage(msg, "", ToolViewCompact, viewportWidth)
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

func ToolCallKeyID(tc types.ToolCall) string {
	if tc.ID != "" {
		return "call:" + tc.ID
	}
	return "call:" + tc.Name + ":" + tc.Arguments
}

func ToolResultKeyID(callID string, toolName string, content string) string {
	if callID != "" {
		return "res:" + callID + ":" + toolName
	}
	return "res:" + toolName + ":" + content
}

func RenderToolCall(tc types.ToolCall, mode ToolViewMode, width int) string {
	renderer := DefaultToolRegistry.Get(tc.Name)
	if mode == ToolViewCompact {
		return renderer.RenderCall(tc, width)
	}
	return renderer.RenderDetailedCall(tc, mode, width)
}

func RenderToolResult(content string, callID string, toolName string, mode ToolViewMode, width int) string {
	renderer := DefaultToolRegistry.fallback
	if toolName != "" {
		renderer = DefaultToolRegistry.Get(toolName)
	}
	var res string
	if mode == ToolViewCompact {
		res = renderer.RenderResult(content, callID, width)
	} else {
		res = renderer.RenderDetailedResult(content, callID, mode, width)
	}
	if res == "" {
		return ""
	}
	return strings.TrimRight(res, "\n") + "\n"
}

func RenderToolCallMessage(msg types.Message, mode ToolViewMode, viewportWidth ...int) string {
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
		lines = append(lines, RenderToolCall(tc, mode, width))
	}
	return strings.Join(lines, "\n")
}

func RenderToolResultMessage(msg types.Message, toolName string, mode ToolViewMode, viewportWidth ...int) string {
	width := 80
	if len(viewportWidth) > 0 && viewportWidth[0] > 0 {
		width = viewportWidth[0]
	}
	return RenderToolResult(msg.Content, msg.ToolCallID, toolName, mode, width)
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
		for _, key := range []string{"path", "command", "query", "url"} {
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
