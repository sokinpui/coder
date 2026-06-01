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
	blockStarts := make(map[int]int)
	selectedBlocks := make(map[int]struct{})

	isVisualState := m.State == stateVisualSelect

	if m.State == stateVisualSelect {
		for i, block := range m.VisualSelect.Blocks {
			blockStarts[block.startIdx] = i
		}
		if m.VisualSelect.Mode == visualModeNone && m.VisualSelect.IsSelecting {
			start, end := m.VisualSelect.Start, m.VisualSelect.Cursor
			if start > end {
				start, end = end, start
			}
			for i := start; i <= end && i < len(m.VisualSelect.Blocks); i++ {
				selectedBlocks[i] = struct{}{}
			}
		}
	}

	for i, msg := range m.Session.GetMessages() {
		messageLineOffsets[i] = currentLine
		var lines []string

		isCursorOn := false
		isSelected := false
		if isVisualState {
			if blockIndex, isStart := blockStarts[i]; isStart {
				isCursorOn = (blockIndex == m.VisualSelect.Cursor)
				switch m.VisualSelect.Mode {
				case visualModeGenerate, visualModeEdit, visualModeBranch:
					isSelected = isCursorOn
				default:
					_, isSelected = selectedBlocks[blockIndex]
				}
			}
		}

		cache, ok := m.Chat.RenderCache[i]
		if ok && cache.content == msg.Content && cache.width == viewportWidth && cache.isVisual == isVisualState && cache.isCursorOn == isCursorOn && cache.isSelected == isSelected {
			lines = cache.lines
		} else {
			renderedMsg := m.renderMessage(msg, viewportWidth, isVisualState, isCursorOn, isSelected)

			if renderedMsg != "" || msg.Type == types.AIMessage {
				lines = strings.Split(renderedMsg, "\n")
				m.Chat.RenderCache[i] = cachedRender{
					lines:      lines,
					content:    msg.Content,
					width:      viewportWidth,
					isVisual:   isVisualState,
					isCursorOn: isCursorOn,
					isSelected: isSelected,
				}
			}
		}

		allLines = append(allLines, lines...)
		currentLine += len(lines)
	}

	if m.State == stateThinking || m.State == stateGenPending {
		text := "Thinking "
		if m.State == stateGenPending {
			text = "Asking "
		}

		// The spinner has its own colors, so we can't render it with the same style as the text.
		thinkingText := thinkingTextStyle.Render(text)

		fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, m.Chat.Spinner.View())
		// Apply padding to the container.
		block := lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
		allLines = append(allLines, strings.Split(block, "\n")...)
	}
	return strings.Join(allLines, "\n"), messageLineOffsets
}

func (m Model) renderMessage(msg types.Message, viewportWidth int, isVisual bool, isCursorOn bool, isSelected bool) string {
	content := msg.Content
	switch msg.Type {
	case types.InitMessage:
		return initMessageStyle.Width(viewportWidth - initMessageStyle.GetHorizontalFrameSize()).Render(content)
	case types.DirectoryMessage:
		return directoryWelcomeStyle.Width(viewportWidth - directoryWelcomeStyle.GetHorizontalFrameSize()).Render(content)
	case types.UserMessage:
		style := userInputStyle
		if isVisual {
			style = applyHighlight(style, isCursorOn, isSelected)
		}
		return style.Width(viewportWidth - style.GetHorizontalFrameSize()).Render(content)
	case types.CommandMessage, types.ShellCmdMessage:
		style := commandInputStyle
		if isVisual {
			style = applyHighlight(style, isCursorOn, isSelected)
		}
		prefix := ""
		if msg.Type == types.ShellCmdMessage { prefix = "Shell: " }
		return style.Width(viewportWidth - style.GetHorizontalFrameSize()).Render(prefix + content)
	case types.ImageMessage:
		style := imageMessageStyle
		if isVisual {
			style = applyHighlight(style, isCursorOn, isSelected)
		}
		return style.Width(viewportWidth - style.GetHorizontalFrameSize()).Render("Image: " + content)
	case types.AIMessage:
		if content == "" {
			return ""
		}
		renderedAI, err := m.GlamourRenderer.Render(content)
		if err != nil {
			renderedAI = content
		}
		if isVisual {
			renderedAI = strings.TrimSpace(renderedAI)
			style := aiVisualBaseStyle
			style = applyHighlight(style, isCursorOn, isSelected)
			return style.Width(viewportWidth - style.GetHorizontalFrameSize()).Render(renderedAI)
		}
		return renderedAI
	case types.CommandResultMessage, types.ShellCmdResultMessage:
		style := commandResultStyle
		if isVisual {
			style = commandResultVisualBaseStyle
			style = applyHighlight(style, isCursorOn, isSelected)
		}
		return style.Width(viewportWidth - style.GetHorizontalFrameSize()).Render(content)
	case types.CommandErrorResultMessage:
		style := commandErrorStyle
		if isVisual {
			style = applyHighlight(style, isCursorOn, isSelected)
		}
		return style.Width(viewportWidth - style.GetHorizontalFrameSize()).Render(content)
	default:
		return ""
	}
}

func (m *Model) renderConversation() string {
	content, offsets := m.renderConversationWithOffsets()
	m.Chat.MessageLineOffsets = offsets
	return content
}
