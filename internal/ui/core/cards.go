package core

import (
	"encoding/json"
	"fmt"
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
		return ImageMessageStyle.Width(viewportWidth - ImageMessageStyle.GetHorizontalFrameSize()).Render("Image: " + content)
	case types.AIMessage:
		if content == "" {
			return ""
		}
		if renderer == nil {
			return content
		}
		renderedAI, err := renderer.Render(content)
		if err != nil {
			return content
		}
		return renderedAI
	case types.ToolCallMessage:
		return RenderToolCallMessage(msg)
	case types.ToolResultMessage:
		return RenderToolResultMessage(msg)
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

func RenderToolCallMessage(msg types.Message) string {
	if len(msg.ToolCalls) == 0 {
		if msg.Content == "" {
			return ""
		}
		return ToolCallStyle.Render("⚡ " + msg.Content)
	}

	var lines []string
	for _, tc := range msg.ToolCalls {
		summary := SummarizeToolArgs(tc.Arguments)
		prefix := ToolCallStyle.Render("⚡ " + tc.Name)
		if summary == "" {
			lines = append(lines, prefix+"()")
			continue
		}
		lines = append(lines, fmt.Sprintf("%s(%s)", prefix, ToolMutedStyle.Render(summary)))
	}
	return strings.Join(lines, "\n")
}

func RenderToolResultMessage(msg types.Message) string {
	output := strings.TrimSpace(msg.Content)
	if output == "" || output == "(no output)" {
		return fmt.Sprintf("↳ %s", ToolMutedStyle.Render("(no output)"))
	}

	firstLine := strings.Split(output, "\n")[0]
	summary := TruncateSingleLine(firstLine, 80)
	if strings.HasPrefix(output, "Error:") || strings.Contains(output, "Command exited with error:") {
		clean := strings.TrimPrefix(summary, "Error: ")
		return fmt.Sprintf("↳ %s %s", ToolErrorStyle.Render("✗"), ToolResultStyle.Render(clean))
	}
	return fmt.Sprintf("↳ %s %s", ToolSuccessStyle.Render("✓"), ToolResultStyle.Render(summary))
}

func SummarizeToolArgs(args string) string {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return ""
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
