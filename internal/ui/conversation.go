package ui

import (
	"strings"

	"coder/internal/types"

	"github.com/charmbracelet/lipgloss"
)

func truncateMessage(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	truncatedLines := lines[:maxLines]
	return strings.Join(truncatedLines, "\n") + "\n... (collapsed)"
}

// renderConversation renders the entire message history.
func (m Model) renderConversation() string {
	var parts []string

	blockStarts := make(map[int]int)
	selectedBlocks := make(map[int]struct{})

	if m.State == stateVisualSelect {
		for i, block := range m.SelectableBlocks {
			blockStarts[block.startIdx] = i
		}

		if m.VisualMode == visualModeNone && m.VisualIsSelecting {
			start, end := m.VisualSelectStart, m.VisualSelectCursor
			if start > end {
				start, end = end, start
			}

			if end < len(m.SelectableBlocks) {
				for i := start; i <= end; i++ {
					selectedBlocks[i] = struct{}{}
				}
			}
		}
	}

	for i, msg := range m.Session.GetMessages() {
		currentMsg := msg // Make a copy to modify content for visual mode
		if m.State == stateVisualSelect {
			switch currentMsg.Type {
			case types.UserMessage, types.AIMessage, types.CommandResultMessage, types.CommandErrorResultMessage:
				currentMsg.Content = truncateMessage(currentMsg.Content, 4)
			}
		}

		var renderedMsg string
		switch currentMsg.Type {
		case types.InitMessage:
			blockWidth := m.Viewport.Width - initMessageStyle.GetHorizontalFrameSize()
			renderedMsg = initMessageStyle.Width(blockWidth).Render(currentMsg.Content)
		case types.DirectoryMessage:
			blockWidth := m.Viewport.Width - directoryWelcomeStyle.GetHorizontalFrameSize()
			renderedMsg = directoryWelcomeStyle.Width(blockWidth).Render(currentMsg.Content)
		case types.UserMessage:
			blockWidth := m.Viewport.Width - userInputStyle.GetHorizontalFrameSize()
			renderedMsg = userInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case types.CommandMessage:
			blockWidth := m.Viewport.Width - commandInputStyle.GetHorizontalFrameSize()
			renderedMsg = commandInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case types.ImageMessage:
			blockWidth := m.Viewport.Width - imageMessageStyle.GetHorizontalFrameSize()
			renderedMsg = imageMessageStyle.Width(blockWidth).Render("Image: " + currentMsg.Content)
		case types.AIMessage:
			if currentMsg.Content == "" {
				continue
			} else {
				renderedAI, err := m.GlamourRenderer.Render(currentMsg.Content)
				if err != nil {
					renderedAI = currentMsg.Content
				}
				renderedMsg = renderedAI
			}
		case types.CommandResultMessage:
			blockWidth := m.Viewport.Width - commandResultStyle.GetHorizontalFrameSize()
			renderedMsg = commandResultStyle.Width(blockWidth).Render(currentMsg.Content)
		case types.CommandErrorResultMessage:
			blockWidth := m.Viewport.Width - commandErrorStyle.GetHorizontalFrameSize()
			renderedMsg = commandErrorStyle.Width(blockWidth).Render(currentMsg.Content)

		}

		if blockIndex, isStart := blockStarts[i]; m.State == stateVisualSelect && isStart {
			isCursorOn := (blockIndex == m.VisualSelectCursor)

			var isSelected bool
			switch m.VisualMode {
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
			renderedMsg = lipgloss.JoinHorizontal(lipgloss.Top, checkbox, renderedMsg)
		}
		parts = append(parts, renderedMsg)
	}

	if m.State == stateThinking || m.State == stateGenPending {
		// The spinner has its own colors, so we can't render it with the same style as the text.
		thinkingText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Render("Thinking ")

		fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, m.Spinner.View())
		// Apply padding to the container.
		block := lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
		parts = append(parts, block)
	}
	return strings.Join(parts, "\n")
}
