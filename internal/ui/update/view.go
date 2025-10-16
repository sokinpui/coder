package update

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
	if len(m.HistoryItems) == 0 {
		return "No history found."
	}

	var b strings.Builder
	b.WriteString("Select a conversation to load:\n\n")

	for i, item := range m.HistoryItems {
		line := fmt.Sprintf("  %s (%s)", item.Title, item.ModifiedAt.Format("2006-01-02 15:04"))
		if i == m.HistoryCussorPos {
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
			case core.UserMessage, core.AIMessage, core.CommandResultMessage, core.CommandErrorResultMessage:
				currentMsg.Content = truncateMessage(currentMsg.Content, 4)
			}
		}

		var renderedMsg string
		switch currentMsg.Type {
		case core.InitMessage:
			blockWidth := m.Viewport.Width - initMessageStyle.GetHorizontalFrameSize()
			renderedMsg = initMessageStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.DirectoryMessage:
			blockWidth := m.Viewport.Width - directoryWelcomeStyle.GetHorizontalFrameSize()
			renderedMsg = directoryWelcomeStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.UserMessage:
			blockWidth := m.Viewport.Width - userInputStyle.GetHorizontalFrameSize()
			renderedMsg = userInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.CommandMessage:
			blockWidth := m.Viewport.Width - commandInputStyle.GetHorizontalFrameSize()
			renderedMsg = commandInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.ImageMessage:
			blockWidth := m.Viewport.Width - imageMessageStyle.GetHorizontalFrameSize()
			renderedMsg = imageMessageStyle.Width(blockWidth).Render("Image: " + currentMsg.Content)
		case core.AIMessage:
			if currentMsg.Content == "" {
				continue
			} else {
				renderedAI, err := m.GlamourRenderer.Render(currentMsg.Content)
				if err != nil {
					renderedAI = currentMsg.Content
				}
				renderedMsg = renderedAI
			}
		case core.CommandResultMessage:
			blockWidth := m.Viewport.Width - commandResultStyle.GetHorizontalFrameSize()
			renderedMsg = commandResultStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.CommandErrorResultMessage:
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

func (m Model) PaletteView() string {
	if !m.ShowPalette || (len(m.PaletteFilteredCommands) == 0 && len(m.PaletteFilteredArguments) == 0) {
		return ""
	}

	var b strings.Builder
	numCommands := len(m.PaletteFilteredCommands)

	if numCommands > 0 {
		b.WriteString(paletteHeaderStyle.Render("Commands"))
		b.WriteString("\n")
		for i, cmd := range m.PaletteFilteredCommands {
			cursorIndex := i
			if cursorIndex == m.PaletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + cmd))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + cmd))
			}
			b.WriteString("\n")
		}
	}

	if numCommands > 0 && len(m.PaletteFilteredArguments) > 0 {
		b.WriteString("\n")
	}

	if len(m.PaletteFilteredArguments) > 0 {
		b.WriteString(paletteHeaderStyle.Render("Arguments"))
		b.WriteString("\n")
		for i, arg := range m.PaletteFilteredArguments {
			cursorIndex := i + numCommands
			if cursorIndex == m.PaletteCursor {
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

func (m Model) StatusView() string {
	if m.StatusBarMessage != "" {
		return statusBarMsgStyle.Render(m.StatusBarMessage)
	}

	if m.CtrlCPressed && m.State == stateIdle && m.TextArea.Value() == "" {
		return statusStyle.Render("Press Ctrl+C again to quit.")
	}

	if m.State == stateHistorySelect {
		return statusStyle.Render("j/k: move | gg/G: top/bottom | enter: load | esc: cancel")
	}

	// Line 1: Title
	var title string
	if m.AnimatingTitle {
		title = m.DisplayedTitle
	} else {
		title = m.Session.GetTitle()
	}
	titlePart := statusBarTitleStyle.MaxWidth(m.Width).Render(title)

	// Line 2: Status
	var leftStatus string
	switch m.State {
	case stateVisualSelect:
		var modeStr, helpStr string
		switch m.VisualMode {
		case visualModeGenerate:
			modeStr = "GENERATE"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		case visualModeEdit:
			modeStr = "EDIT"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		case visualModeBranch:
			modeStr = "BRANCH"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		default: // visualModeNone
			modeStr = "VISUAL"
			if m.VisualIsSelecting {
				helpStr = "j/k: move | y: copy | d: delete | esc: cancel selection"
			} else {
				helpStr = "j/k: move | v: start selection | esc: cancel"
			}
		}
		leftStatus = statusStyle.Render(fmt.Sprintf("-- %s MODE -- | %s", modeStr, helpStr))
	case stateCancelling:
		leftStatus = generatingStatusStyle.Render("Cancelling...")
	}

	modeInfo := fmt.Sprintf("Mode: %s", m.Session.GetConfig().AppMode)
	modelInfo := fmt.Sprintf("Model: %s", m.Session.GetConfig().Generation.ModelCode)

	var tokenInfo string
	if m.IsCountingTokens {
		tokenInfo = "Tokens: counting..."
	} else if m.TokenCount > 0 {
		tokenInfo = fmt.Sprintf("Tokens: %d", m.TokenCount)
	}

	modePart := modelInfoStyle.Render(modeInfo)
	modelPart := modelInfoStyle.Render(modelInfo)
	tokenPart := tokenCountStyle.Render(tokenInfo)

	rightStatusItems := []string{}
	if m.State != stateVisualSelect {
		if tokenPart != "" {
			rightStatusItems = append(rightStatusItems, tokenPart)
		}
		rightStatusItems = append(rightStatusItems, modePart, modelPart)
	}

	if m.State == stateGenPending || m.State == stateThinking || m.State == stateGenerating {
		statusText := "Thinking" // Default for genpending and thinking
		if m.State == stateGenerating {
			statusText = "Generating"
		}
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, statusStyle.Render(statusText+" "), m.Spinner.View())
		rightStatusItems = append(rightStatusItems, spinnerWithText)
	}
	rightStatus := strings.Join(rightStatusItems, " | ")

	var statusLine string
	if leftStatus != "" {
		spacing := m.Width - lipgloss.Width(leftStatus) - lipgloss.Width(rightStatus)
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
	if m.Quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.Viewport.View())
	b.WriteString("\n")

	if m.State != stateHistorySelect {
		b.WriteString(textAreaStyle.Render(m.TextArea.View()))
		b.WriteString("\n")
	}
	b.WriteString(m.StatusView())

	return b.String()
}
