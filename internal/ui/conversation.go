package ui

import (
	"strings"

	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/lipgloss"
)

// renderConversationWithOffsets renders the conversation and returns the content string
// and a map of message index to its starting line number.
func (m Model) renderConversationWithOffsets() (string, map[int]int) {
	messageLineOffsets := make(map[int]int)
	viewportWidth := m.Chat.Viewport.Width
	currentLine := 0
	var allLines []string

	for i, msg := range m.Session.GetMessages() {
		messageLineOffsets[i] = currentLine
		var lines []string

		cache, ok := m.Chat.RenderCache[i]
		if ok && cache.content == msg.Content && cache.width == viewportWidth {
			lines = cache.lines
		} else {
			renderedMsg := m.renderMessage(msg, viewportWidth)

			if renderedMsg != "" || msg.Type == types.AIMessage {
				lines = strings.Split(renderedMsg, "\n")
				m.Chat.RenderCache[i] = cachedRender{
					lines:   lines,
					content: msg.Content,
					width:   viewportWidth,
				}
			}
		}

		allLines = append(allLines, lines...)
		currentLine += len(lines)
	}

	if m.State == stateAsking || m.State == stateThinking {
		thinkingLine := m.renderThinkingLine()
		allLines = append(allLines, strings.Split(thinkingLine, "\n")...)
	}
	return strings.Join(allLines, "\n"), messageLineOffsets
}

func (m Model) renderMessage(msg types.Message, viewportWidth int) string {
	content := msg.Content
	switch msg.Type {
	case types.InitMessage:
		return initMessageStyle.Width(viewportWidth - initMessageStyle.GetHorizontalFrameSize()).Render(content)
	case types.DirectoryMessage:
		return directoryWelcomeStyle.Width(viewportWidth - directoryWelcomeStyle.GetHorizontalFrameSize()).Render(content)
	case types.UserMessage:
		return userInputStyle.Width(viewportWidth - userInputStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandMessage, types.ShellCmdMessage:
		prefix := ""
		if msg.Type == types.ShellCmdMessage {
			prefix = "Shell: "
		}
		return commandInputStyle.Width(viewportWidth - commandInputStyle.GetHorizontalFrameSize()).Render(prefix + content)
	case types.ImageMessage:
		return imageMessageStyle.Width(viewportWidth - imageMessageStyle.GetHorizontalFrameSize()).Render("Image: " + content)
	case types.AIMessage:
		if content == "" {
			return ""
		}
		renderedAI, err := m.GlamourRenderer.Render(content)
		if err != nil {
			return content
		}
		return renderedAI
	case types.CommandResultMessage, types.ShellCmdResultMessage:
		return commandResultStyle.Width(viewportWidth - commandResultStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandErrorResultMessage:
		return commandErrorStyle.Width(viewportWidth - commandErrorStyle.GetHorizontalFrameSize()).Render(content)
	default:
		return ""
	}
}

func (m Model) renderThinkingLine() string {
	text := "Thinking "
	if m.State == stateAsking {
		text = "Asking "
	}
	thinkingText := thinkingTextStyle.Render(text)
	fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, m.Chat.Spinner.View())
	return lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
}

func (m *Model) renderConversation() string {
	content, offsets := m.renderConversationWithOffsets()
	m.Chat.MessageLineOffsets = offsets
	return content
}
