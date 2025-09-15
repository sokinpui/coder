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

// renderConversation renders the entire message history.
func (m Model) renderConversation() string {
	var parts []string

	selectedIndices := make(map[int]struct{})
	if m.state == stateVisualSelect {
		if m.visualMode == visualModeGenerate || m.visualMode == visualModeEdit {
			if m.visualSelectCursor < len(m.selectableBlocks) {
				block := m.selectableBlocks[m.visualSelectCursor]
				for j := block.startIdx; j <= block.endIdx; j++ {
					selectedIndices[j] = struct{}{}
				}
			}
		} else {
			start, end := m.visualSelectStart, m.visualSelectCursor
			if start > end {
				start, end = end, start
			}

			if end < len(m.selectableBlocks) {
				for i := start; i <= end; i++ {
					block := m.selectableBlocks[i]
					for j := block.startIdx; j <= block.endIdx; j++ {
						selectedIndices[j] = struct{}{}
					}
				}
			}
		}
	}

	for i, msg := range m.session.GetMessages() {
		currentMsg := msg // Make a copy to modify content for visual mode
		if m.state == stateVisualSelect {
			switch currentMsg.Type {
			case core.UserMessage, core.AIMessage, core.ActionResultMessage, core.CommandResultMessage, core.ActionErrorResultMessage, core.CommandErrorResultMessage:
				currentMsg.Content = truncateMessage(currentMsg.Content, 4)
			}
		}

		var renderedMsg string
		switch currentMsg.Type {
		case core.InitMessage:
			blockWidth := m.viewport.Width - initMessageStyle.GetHorizontalPadding()
			renderedMsg = initMessageStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.DirectoryMessage:
			blockWidth := m.viewport.Width - directoryWelcomeStyle.GetHorizontalPadding()
			renderedMsg = directoryWelcomeStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.UserMessage:
			blockWidth := m.viewport.Width - userInputStyle.GetHorizontalPadding()
			renderedMsg = userInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.ActionMessage:
			blockWidth := m.viewport.Width - actionInputStyle.GetHorizontalPadding()
			renderedMsg = actionInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.CommandMessage:
			blockWidth := m.viewport.Width - commandInputStyle.GetHorizontalPadding()
			renderedMsg = commandInputStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.AIMessage:
			if currentMsg.Content == "" {
				renderedMsg = ""
			} else {
				renderedAI, err := m.glamourRenderer.Render(currentMsg.Content)
				if err != nil {
					renderedAI = currentMsg.Content
				}
				renderedMsg = renderedAI
			}
		case core.ActionResultMessage:
			blockWidth := m.viewport.Width - actionResultStyle.GetHorizontalPadding()
			renderedMsg = actionResultStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.CommandResultMessage:
			// Don't render the special result messages that trigger visual modes.
			if currentMsg.Content == core.CopyModeResult || currentMsg.Content == core.DeleteModeResult || currentMsg.Content == core.GenerateModeResult || currentMsg.Content == core.EditModeResult {
				continue
			}
			blockWidth := m.viewport.Width - commandResultStyle.GetHorizontalPadding()
			renderedMsg = commandResultStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.ActionErrorResultMessage:
			blockWidth := m.viewport.Width - actionErrorStyle.GetHorizontalPadding()
			renderedMsg = actionErrorStyle.Width(blockWidth).Render(currentMsg.Content)
		case core.CommandErrorResultMessage:
			blockWidth := m.viewport.Width - commandErrorStyle.GetHorizontalPadding()
			renderedMsg = commandErrorStyle.Width(blockWidth).Render(currentMsg.Content)
		}

		if _, isSelected := selectedIndices[i]; isSelected {
			renderedMsg = visualSelectStyle.Render(renderedMsg)
		}
		parts = append(parts, renderedMsg)
	}

	if m.state == stateThinking {
		thinkingMsg := fmt.Sprintf("%s AI is thinking...", m.spinner.View())
		block := thinkingStyle.Render(thinkingMsg)
		parts = append(parts, block)
	}
	return strings.Join(parts, "\n")
}

func (m Model) paletteView() string {
	if !m.showPalette || (len(m.paletteFilteredActions) == 0 && len(m.paletteFilteredCommands) == 0 && len(m.paletteFilteredArguments) == 0) {
		return ""
	}

	var b strings.Builder
	numActions := len(m.paletteFilteredActions)
	numCommands := len(m.paletteFilteredCommands)

	if numActions > 0 {
		b.WriteString(paletteHeaderStyle.Render("Actions"))
		b.WriteString("\n")
		for i, action := range m.paletteFilteredActions {
			if i == m.paletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + action))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + action))
			}
			b.WriteString("\n")
		}
	}

	if numActions > 0 && numCommands > 0 {
		b.WriteString("\n")
	}

	if numCommands > 0 {
		b.WriteString(paletteHeaderStyle.Render("Commands"))
		b.WriteString("\n")
		for i, cmd := range m.paletteFilteredCommands {
			cursorIndex := i + numActions
			if cursorIndex == m.paletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + cmd))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + cmd))
			}
			b.WriteString("\n")
		}
	}

	if (numActions > 0 || numCommands > 0) && len(m.paletteFilteredArguments) > 0 {
		b.WriteString("\n")
	}

	if len(m.paletteFilteredArguments) > 0 {
		b.WriteString(paletteHeaderStyle.Render("Arguments"))
		b.WriteString("\n")
		for i, arg := range m.paletteFilteredArguments {
			cursorIndex := i + numActions + numCommands
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

	var status string
	var modeStr string
	switch m.state {
	case stateThinking, stateGenerating:
		status = "Ctrl+U/D: scroll | Ctrl+C: cancel"
	case stateCancelling:
		return generatingStatusStyle.Render("Cancelling...")
	case stateVisualSelect:
		modeStr = "COPY"
		if m.visualMode == visualModeDelete {
			modeStr = "DELETE"
		} else if m.visualMode == visualModeGenerate {
			modeStr = "GENERATE"
		} else if m.visualMode == visualModeEdit {
			modeStr = "EDIT"
		}
		status = fmt.Sprintf("-- %s MODE -- | j/k: move | enter: confirm | esc: cancel", modeStr)
	default: // stateIdle
		status = ""
	}

	modeInfo := fmt.Sprintf("Mode: %s", m.session.GetConfig().AppMode)
	modelInfo := fmt.Sprintf("Model: %s", m.session.GetConfig().Generation.ModelCode)

	var tokenInfo string
	if m.isCountingTokens {
		tokenInfo = "Tokens: counting..."
	} else if m.tokenCount > 0 {
		tokenInfo = fmt.Sprintf("Tokens: %d", m.tokenCount)
	}

	statusPart := statusStyle.Render(status)
	modePart := modelInfoStyle.Render(modeInfo)
	modelPart := modelInfoStyle.Render(modelInfo)
	tokenPart := tokenCountStyle.Render(tokenInfo)

	rightSide := lipgloss.JoinHorizontal(lipgloss.Top, modePart, " | ", modelPart)
	if tokenPart != "" {
		rightSide = lipgloss.JoinHorizontal(lipgloss.Top, tokenPart, " | ", rightSide)
	}

	spacing := m.width - lipgloss.Width(statusPart) - lipgloss.Width(rightSide)
	if spacing < 1 {
		return statusPart
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, statusPart, strings.Repeat(" ", spacing), rightSide)
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	if m.showPalette {
		b.WriteString(m.paletteView())
		b.WriteString("\n")
	}

	b.WriteString(textAreaStyle.Render(m.textArea.View()))
	b.WriteString("\n")
	b.WriteString(m.statusView())

	return b.String()
}
