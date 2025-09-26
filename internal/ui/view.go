package ui

import (
	"fmt"
	"strings"

	"coder/internal/core"

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

func (m Model) historyView() string {
	if len(m.historyItems) == 0 {
		return "No history found."
	}

	var b strings.Builder
	b.WriteString("Select a conversation to load:\n\n")

	for i, item := range m.historyItems {
		line := fmt.Sprintf("  %s (%s)", item.Title, item.ModifiedAt.Format("2006-01-02 15:04"))
		if i == m.historySelectCursor {
			b.WriteString(paletteSelectedItemStyle.Render("▸" + line))
		} else {
			b.WriteString(paletteItemStyle.Render(" " + line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderConversation renders the entire message history.
func (m Model) renderConversation() string {
	var parts []string

	blockStarts := make(map[int]int)
	selectedBlocks := make(map[int]struct{})

	if m.state == stateVisualSelect {
		for i, block := range m.selectableBlocks {
			blockStarts[block.startIdx] = i
		}

		if m.visualMode == visualModeNone && m.visualIsSelecting {
			start, end := m.visualSelectStart, m.visualSelectCursor
			if start > end {
				start, end = end, start
			}

			if end < len(m.selectableBlocks) {
				for i := start; i <= end; i++ {
					selectedBlocks[i] = struct{}{}
				}
			}
		}
	}

	for i, msg := range m.session.GetMessages() {
		currentMsg := msg // Make a copy to modify content for visual mode
		if m.state == stateVisualSelect {
			switch currentMsg.Type {
			case core.UserMessage, core.AIMessage, core.CommandResultMessage, core.CommandErrorResultMessage:
				currentMsg.Content = truncateMessage(currentMsg.Content, 4)
			}
		}

		var renderedMsg string
		switch currentMsg.Type {
		case core.InitMessage:
			blockWidth := m.viewport.Width - initMessageStyle.GetHorizontalFrameSize()
			renderedMsg = initMessageStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.DirectoryMessage:
			blockWidth := m.viewport.Width - directoryWelcomeStyle.GetHorizontalFrameSize()
			renderedMsg = directoryWelcomeStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.UserMessage:
			blockWidth := m.viewport.Width - userInputStyle.GetHorizontalFrameSize()
			renderedMsg = userInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.CommandMessage:
			blockWidth := m.viewport.Width - commandInputStyle.GetHorizontalFrameSize()
			renderedMsg = commandInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.AIMessage:
			if currentMsg.Content == "" {
				continue
			} else {
				renderedAI, err := m.glamourRenderer.Render(currentMsg.Content)
				if err != nil {
					renderedAI = currentMsg.Content
				}
				renderedMsg = renderedAI
			}
		case core.CommandResultMessage:
			blockWidth := m.viewport.Width - commandResultStyle.GetHorizontalFrameSize()
			renderedMsg = commandResultStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.CommandErrorResultMessage:
			blockWidth := m.viewport.Width - commandErrorStyle.GetHorizontalFrameSize()
			renderedMsg = commandErrorStyle.Width(blockWidth).Render(currentMsg.Content)
		}

		if blockIndex, isStart := blockStarts[i]; m.state == stateVisualSelect && isStart {
			isCursorOn := (blockIndex == m.visualSelectCursor)

			var isSelected bool
			if m.visualMode == visualModeGenerate || m.visualMode == visualModeEdit || m.visualMode == visualModeBranch {
				isSelected = isCursorOn
			} else { // visualModeNone
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

	if m.state == stateThinking {
		// The spinner has its own colors, so we can't render it with the same style as the text.
		thinkingText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true).
			Render("Thinking ")

		fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, m.spinner.View())
		// Apply padding to the container.
		block := lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
		parts = append(parts, block)
	}
	return strings.Join(parts, "\n")
}

func (m Model) paletteView() string {
	if !m.showPalette || (len(m.paletteFilteredCommands) == 0 && len(m.paletteFilteredArguments) == 0) {
		return ""
	}

	var b strings.Builder
	numCommands := len(m.paletteFilteredCommands)

	if numCommands > 0 {
		b.WriteString(paletteHeaderStyle.Render("Commands"))
		b.WriteString("\n")
		for i, cmd := range m.paletteFilteredCommands {
			cursorIndex := i
			if cursorIndex == m.paletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + cmd))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + cmd))
			}
			b.WriteString("\n")
		}
	}

	if numCommands > 0 && len(m.paletteFilteredArguments) > 0 {
		b.WriteString("\n")
	}

	if len(m.paletteFilteredArguments) > 0 {
		b.WriteString(paletteHeaderStyle.Render("Arguments"))
		b.WriteString("\n")
		for i, arg := range m.paletteFilteredArguments {
			cursorIndex := i + numCommands
			if cursorIndex == m.paletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + arg))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + arg))
			}
			b.WriteString("\n")
		}
	}

	// Trim trailing newline
	content := strings.TrimRight(b.String(), "\n")

	return paletteContainerStyle.Render(content)
}

func (m Model) statusView() string {
	if m.statusBarMessage != "" {
		return statusBarMsgStyle.Render(m.statusBarMessage)
	}

	if m.ctrlCPressed && m.state == stateIdle && m.textArea.Value() == "" {
		return statusStyle.Render("Press Ctrl+C again to quit.")
	}

	if m.state == stateHistorySelect {
		return statusStyle.Render("j/k: move | enter: load | esc: cancel")
	}

	// Line 1: Title
	var title string
	if m.animatingTitle {
		title = m.displayedTitle
	} else {
		title = m.session.GetTitle()
	}
	titlePart := statusBarTitleStyle.MaxWidth(m.width).Render(title)

	// Line 2: Status
	var leftStatus string
	if m.state == stateVisualSelect {
		var modeStr string
		var helpStr string
		if m.visualMode == visualModeGenerate {
			modeStr = "GENERATE"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		} else if m.visualMode == visualModeEdit {
			modeStr = "EDIT"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		} else if m.visualMode == visualModeBranch {
			modeStr = "BRANCH"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		} else { // visualModeNone
			modeStr = "VISUAL"
			if m.visualIsSelecting {
				helpStr = "j/k: move | y: copy | d: delete | esc: cancel selection"
			} else {
				helpStr = "j/k: move | v: start selection | esc: cancel"
			}
		}
		leftStatus = statusStyle.Render(fmt.Sprintf("-- %s MODE -- | %s", modeStr, helpStr))
	} else if m.state == stateCancelling {
		leftStatus = generatingStatusStyle.Render("Cancelling...")
	}

	modeInfo := fmt.Sprintf("Mode: %s", m.session.GetConfig().AppMode)
	modelInfo := fmt.Sprintf("Model: %s", m.session.GetConfig().Generation.ModelCode)

	var tokenInfo string
	if m.isCountingTokens {
		tokenInfo = "Tokens: counting..."
	} else if m.tokenCount > 0 {
		tokenInfo = fmt.Sprintf("Tokens: %d", m.tokenCount)
	}

	modePart := modelInfoStyle.Render(modeInfo)
	modelPart := modelInfoStyle.Render(modelInfo)
	tokenPart := tokenCountStyle.Render(tokenInfo)

	rightStatusItems := []string{}
	if tokenPart != "" {
		rightStatusItems = append(rightStatusItems, tokenPart)
	}
	rightStatusItems = append(rightStatusItems, modePart, modelPart)

	if m.state == stateThinking {
		rightStatusItems = append(rightStatusItems, statusStyle.Render("Thinking..."))
	} else if m.state == stateGenerating {
		rightStatusItems = append(rightStatusItems, statusStyle.Render("Generating..."))
	}
	rightStatus := strings.Join(rightStatusItems, " | ")

	var statusLine string
	if leftStatus != "" {
		spacing := m.width - lipgloss.Width(leftStatus) - lipgloss.Width(rightStatus)
		if spacing < 1 {
			spacing = 1
		}
		statusLine = lipgloss.JoinHorizontal(lipgloss.Top, leftStatus, strings.Repeat(" ", spacing), rightStatus)
	} else {
		statusLine = rightStatus
	}

	return lipgloss.JoinVertical(lipgloss.Left, titlePart, statusLine)
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	if m.state != stateHistorySelect {
		if m.showPalette {
			b.WriteString(m.paletteView())
			b.WriteString("\n")
		}

		b.WriteString(textAreaStyle.Render(m.textArea.View()))
		b.WriteString("\n")
	}
	b.WriteString(m.statusView())

	return b.String()
}
