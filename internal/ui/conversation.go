package ui

import (
	"regexp"
	"strings"

	"github.com/sokinpui/coder/internal/types"

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

func (m Model) highlightMatches(text string) string {
	if m.Chat.SearchQuery == "" {
		return text
	}

	// Case-insensitive regex for the search query
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(m.Chat.SearchQuery))
	if err != nil {
		return text
	}

	return re.ReplaceAllStringFunc(text, func(match string) string {
		return searchHighlightStyle.Render(match)
	})
}

// renderConversationWithOffsets renders the conversation and returns the content string
// and a map of message index to its starting line number.
func (m Model) renderConversationWithOffsets() (string, map[int]int) {
	messageLineOffsets := make(map[int]int)
	currentLine := 0
	var parts []string

	blockStarts := make(map[int]int)
	selectedBlocks := make(map[int]struct{})

	if m.State == stateVisualSelect {
		for i, block := range m.VisualSelect.Blocks {
			blockStarts[block.startIdx] = i
		}

		if m.VisualSelect.Mode == visualModeNone && m.VisualSelect.IsSelecting {
			start, end := m.VisualSelect.Start, m.VisualSelect.Cursor
			if start > end {
				start, end = end, start
			}

			if end < len(m.VisualSelect.Blocks) {
				for i := start; i <= end; i++ {
					selectedBlocks[i] = struct{}{}
				}
			}
		}
	}

	for i, msg := range m.Session.GetMessages() {
		messageLineOffsets[i] = currentLine
		currentMsg := msg // Make a copy to modify content for visual mode
		if m.State == stateVisualSelect {
			switch currentMsg.Type {
			case types.UserMessage, types.AIMessage, types.CommandResultMessage, types.CommandErrorResultMessage:
				currentMsg.Content = truncateMessage(currentMsg.Content, 4)
			}
		}

		contentToRender := currentMsg.Content
		if m.Chat.SearchQuery != "" {
			switch currentMsg.Type {
			case types.UserMessage, types.CommandMessage, types.CommandResultMessage, types.CommandErrorResultMessage:
				// Highlight text before applying Lipgloss borders/styles
				contentToRender = m.highlightMatches(contentToRender)
			}
		}

		var renderedMsg string
		switch currentMsg.Type {
		case types.InitMessage:
			blockWidth := m.Chat.Viewport.Width - initMessageStyle.GetHorizontalFrameSize()
			renderedMsg = initMessageStyle.Width(blockWidth).Render(currentMsg.Content)
		case types.DirectoryMessage:
			blockWidth := m.Chat.Viewport.Width - directoryWelcomeStyle.GetHorizontalFrameSize()
			renderedMsg = directoryWelcomeStyle.Width(blockWidth).Render(currentMsg.Content)
		case types.UserMessage:
			blockWidth := m.Chat.Viewport.Width - userInputStyle.GetHorizontalFrameSize()
			renderedMsg = userInputStyle.Width(blockWidth).Render(contentToRender)
		case types.CommandMessage:
			blockWidth := m.Chat.Viewport.Width - commandInputStyle.GetHorizontalFrameSize()
			renderedMsg = commandInputStyle.Width(blockWidth).Render(contentToRender)
		case types.ImageMessage:
			blockWidth := m.Chat.Viewport.Width - imageMessageStyle.GetHorizontalFrameSize()
			renderedMsg = imageMessageStyle.Width(blockWidth).Render("Image: " + currentMsg.Content)
		case types.AIMessage:
			if contentToRender == "" {
				continue
			} else {
				renderedAI, err := m.GlamourRenderer.Render(contentToRender)
				if err != nil {
					renderedAI = contentToRender
				}
				renderedMsg = m.highlightMatches(renderedAI)
			}
		case types.CommandResultMessage:
			blockWidth := m.Chat.Viewport.Width - commandResultStyle.GetHorizontalFrameSize()
			renderedMsg = commandResultStyle.Width(blockWidth).Render(contentToRender)
		case types.CommandErrorResultMessage:
			blockWidth := m.Chat.Viewport.Width - commandErrorStyle.GetHorizontalFrameSize()
			renderedMsg = commandErrorStyle.Width(blockWidth).Render(contentToRender)

		}

		if i == m.Chat.SearchFocusMsgIndex {
			lines := strings.Split(renderedMsg, "\n")
			// Adjust for potential top border offset in styled blocks
			borderOffset := 0
			switch currentMsg.Type {
			case types.UserMessage, types.CommandMessage, types.ImageMessage:
				borderOffset = 1
			}

			targetLine := m.Chat.SearchFocusLineNum + borderOffset
			indicatorStr := "▸ "
			indicator := searchIndicatorStyle.Render(indicatorStr)
			spacer := strings.Repeat(" ", lipgloss.Width(indicatorStr))

			for l := range lines {
				if l == targetLine {
					lines[l] = indicator + lines[l]
				} else {
					lines[l] = spacer + lines[l]
				}
			}
			renderedMsg = strings.Join(lines, "\n")
		}

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
			renderedMsg = lipgloss.JoinHorizontal(lipgloss.Top, checkbox, renderedMsg)
		}
		parts = append(parts, renderedMsg)
		currentLine += strings.Count(renderedMsg, "\n") + 1
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
		parts = append(parts, block)
	}
	return strings.Join(parts, "\n"), messageLineOffsets
}

// renderConversation renders the entire message history.
func (m Model) renderConversation() string {
	content, _ := m.renderConversationWithOffsets()
	return content
}
