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

		cache, ok := m.Chat.RenderCache[i]
		if ok && cache.content == msg.Content && cache.width == viewportWidth && cache.isVisual == isVisualState {
			lines = cache.lines
		} else {
			renderedMsg := m.renderMessage(msg, viewportWidth, isVisualState)

			if renderedMsg != "" || msg.Type == types.AIMessage {
				lines = strings.Split(renderedMsg, "\n")
				m.Chat.RenderCache[i] = cachedRender{
					lines:    lines,
					content:  msg.Content,
					width:    viewportWidth,
					isVisual: isVisualState,
				}
			}
		}

		if len(lines) == 0 && msg.Type == types.AIMessage {
			continue
		}

		// Post-processing: Search Highlighting and Selection UI
		// We only apply search highlighting to lines within or near the viewport.
		processedLines := make([]string, len(lines))
		copy(processedLines, lines)

		if blockIndex, isStart := blockStarts[i]; m.State == stateVisualSelect && isStart {
			isCursorOn := (blockIndex == m.VisualSelect.Cursor)

			var isSelected bool
			switch m.VisualSelect.Mode {
			case visualModeGenerate, visualModeEdit, visualModeBranch:
				isSelected = isCursorOn
			default: // visualModeNone
				_, isSelected = selectedBlocks[blockIndex]
			}

			var checkbox string
			if isSelected {
				checkbox = "[x] "
			} else {
				checkbox = "[ ] "
			}

			if isCursorOn {
				checkbox = "▸ " + checkbox
			} else {
				checkbox = "  " + checkbox
			}

			if isCursorOn {
				checkbox = paletteSelectedItemStyle.Render(checkbox)
			} else {
				checkbox = paletteItemStyle.Render(checkbox)
			}
			for l, line := range processedLines {
				if l == 0 {
					processedLines[l] = lipgloss.JoinHorizontal(lipgloss.Top, checkbox, line)
				} else {
					processedLines[l] = strings.Repeat(" ", lipgloss.Width(checkbox)) + line
				}
			}
		}

		allLines = append(allLines, processedLines...)
		currentLine += len(processedLines)
	}

	if m.State == stateThinking || m.State == stateGenPending {
		// The spinner has its own colors, so we can't render it with the same style as the text.
		thinkingText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Render("Thinking ")

		fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, m.Chat.Spinner.View())
		// Apply padding to the container.
		block := lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
		allLines = append(allLines, strings.Split(block, "\n")...)
	}
	return strings.Join(allLines, "\n"), messageLineOffsets
}

func (m Model) renderMessage(msg types.Message, viewportWidth int, isVisual bool) string {
	content := msg.Content
	switch msg.Type {
	case types.InitMessage:
		return initMessageStyle.Width(viewportWidth - initMessageStyle.GetHorizontalFrameSize()).Render(content)
	case types.DirectoryMessage:
		return directoryWelcomeStyle.Width(viewportWidth - directoryWelcomeStyle.GetHorizontalFrameSize()).Render(content)
	case types.UserMessage:
		return userInputStyle.Width(viewportWidth - userInputStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandMessage:
		return commandInputStyle.Width(viewportWidth - commandInputStyle.GetHorizontalFrameSize()).Render(content)
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
	case types.CommandResultMessage:
		return commandResultStyle.Width(viewportWidth - commandResultStyle.GetHorizontalFrameSize()).Render(content)
	case types.CommandErrorResultMessage:
		return commandErrorStyle.Width(viewportWidth - commandErrorStyle.GetHorizontalFrameSize()).Render(content)
	default:
		return ""
	}
}

func (m *Model) renderConversation() string {
	content, offsets := m.renderConversationWithOffsets()
	m.Chat.MessageLineOffsets = offsets
	return content
}
