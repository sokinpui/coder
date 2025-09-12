package ui

import (
	"coder/internal/core"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

// renderConversation renders the entire message history.
func (m Model) renderConversation() string {
	var parts []string
	for _, msg := range m.messages {
		switch msg.Type {
		case core.InitMessage:
			blockWidth := m.viewport.Width - initMessageStyle.GetHorizontalPadding()
			block := initMessageStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, block)
		case core.UserMessage:
			blockWidth := m.viewport.Width - userInputStyle.GetHorizontalPadding()
			block := userInputStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, block)
		case core.ActionMessage:
			blockWidth := m.viewport.Width - actionInputStyle.GetHorizontalPadding()
			block := actionInputStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, block)
		case core.CommandMessage:
			blockWidth := m.viewport.Width - commandInputStyle.GetHorizontalPadding()
			block := commandInputStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, block)
		case core.AIMessage:
			var content string
			if msg.Content != "" {
				renderedAI, err := m.glamourRenderer.Render(msg.Content)
				if err != nil {
					renderedAI = msg.Content
				}
				content = renderedAI
			}
			parts = append(parts, content)
		case core.ActionResultMessage:
			blockWidth := m.viewport.Width - actionResultStyle.GetHorizontalPadding()
			cmdResultBlock := actionResultStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, cmdResultBlock)
		case core.CommandResultMessage:
			blockWidth := m.viewport.Width - commandResultStyle.GetHorizontalPadding()
			cmdResultBlock := commandResultStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, cmdResultBlock)
		case core.ActionErrorResultMessage:
			blockWidth := m.viewport.Width - actionErrorStyle.GetHorizontalPadding()
			cmdErrorBlock := actionErrorStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, cmdErrorBlock)
		case core.CommandErrorResultMessage:
			blockWidth := m.viewport.Width - commandErrorStyle.GetHorizontalPadding()
			cmdErrorBlock := commandErrorStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, cmdErrorBlock)
		}
	}

	if m.state == stateThinking {
		thinkingMsg := fmt.Sprintf("%s AI is thinking...", m.spinner.View())
		block := thinkingStyle.Render(thinkingMsg)
		parts = append(parts, block)
	}
	return strings.Join(parts, "\n")
}

func (m Model) paletteView() string {
	if !m.showPalette || (len(m.paletteFilteredActions) == 0 && len(m.paletteFilteredCommands) == 0) {
		return ""
	}

	var b strings.Builder
	numActions := len(m.paletteFilteredActions)

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

	if numActions > 0 && len(m.paletteFilteredCommands) > 0 {
		b.WriteString("\n")
	}

	if len(m.paletteFilteredCommands) > 0 {
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

	// Trim trailing newline
	content := strings.TrimRight(b.String(), "\n")

	return paletteContainerStyle.Render(content)
}

func (m Model) helpView() string {
	if m.ctrlCPressed && m.state == stateIdle && m.textArea.Value() == "" {
		return helpStyle.Render("Press Ctrl+C again to quit.")
	}

	var help string
	switch m.state {
	case stateThinking, stateGenerating:
		help = "Ctrl+U/D: scroll | Ctrl+C: cancel"
	case stateCancelling:
		return generatingHelpStyle.Render("Cancelling...")
	default: // stateIdle
		help = "Esc: clear | Ctrl+C: clear/quit"
	}

	modelInfo := fmt.Sprintf("Model: %s", m.config.Generation.ModelCode)

	var tokenInfo string
	if m.isCountingTokens {
		tokenInfo = "Tokens: counting..."
	} else if m.tokenCount > 0 {
		tokenInfo = fmt.Sprintf("Tokens: %d", m.tokenCount)
	}

	helpPart := helpStyle.Render(help)
	modelPart := modelInfoStyle.Render(modelInfo)
	tokenPart := tokenCountStyle.Render(tokenInfo)

	rightSide := modelPart
	if tokenPart != "" {
		rightSide = lipgloss.JoinHorizontal(lipgloss.Top, tokenPart, " | ", modelPart)
	}

	spacing := m.width - lipgloss.Width(helpPart) - lipgloss.Width(rightSide)
	if spacing < 1 {
		return helpPart
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, helpPart, strings.Repeat(" ", spacing), rightSide)
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
	b.WriteString(m.helpView())

	return b.String()
}
