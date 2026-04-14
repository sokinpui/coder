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
	viewportWidth := m.Chat.Viewport.Width
	currentLine := 0
	var allLines []string
	blockStarts := make(map[int]int)
	selectedBlocks := make(map[int]struct{})

	isVisualState := m.State == stateVisualSelect
	viewportTop := m.Chat.Viewport.YOffset
	viewportBottom := viewportTop + m.Chat.Viewport.Height

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

		if i == m.Chat.SearchFocusMsgIndex {
			borderOffset := 0
			switch msg.Type {
			case types.UserMessage, types.CommandMessage, types.ImageMessage:
				borderOffset = 1
			}

			targetLine := m.Chat.SearchFocusLineNum + borderOffset
			indicatorStr := "▸ "
			indicator := searchIndicatorStyle.Render(indicatorStr)
			spacer := strings.Repeat(" ", lipgloss.Width(indicatorStr))

			for l := range processedLines {
				if l == targetLine {
					processedLines[l] = indicator + processedLines[l]
				} else {
					processedLines[l] = spacer + processedLines[l]
				}
			}
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
			for l, line := range processedLines {
				if l == 0 {
					processedLines[l] = lipgloss.JoinHorizontal(lipgloss.Top, checkbox, line)
				} else {
					processedLines[l] = strings.Repeat(" ", lipgloss.Width(checkbox)) + line
				}
			}
		}

		if m.Chat.SearchQuery != "" {
			for l, line := range processedLines {
				absLine := currentLine + l
				if absLine >= viewportTop-10 && absLine <= viewportBottom+10 {
					processedLines[l] = m.highlightMatches(line)
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
	currentMsg := msg
	if isVisual {
		switch currentMsg.Type {
		case types.UserMessage, types.AIMessage, types.CommandResultMessage, types.CommandErrorResultMessage:
			currentMsg.Content = truncateMessage(currentMsg.Content, 4)
		}
	}

	content := currentMsg.Content
	switch currentMsg.Type {
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

func (m Model) renderConversation() string {
	content, _ := m.renderConversationWithOffsets()
	return content
}
