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
	return strings.Join(parts, "\n")
}

func (m Model) paletteView() string {
	if !m.showPalette || len(m.paletteItems) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(paletteHeaderStyle.Render("Suggestions"))
	b.WriteString("\n")

	for i, item := range m.paletteItems {
		if i == m.paletteCursor {
			b.WriteString(paletteSelectedItemStyle.Render("▸ " + item))
		} else {
			b.WriteString(paletteItemStyle.Render("  " + item))
		}
		b.WriteString("\n")
	}

	// Trim trailing newline
	content := strings.TrimRight(b.String(), "\n")

	return paletteContainerStyle.Render(content)
}

func (m Model) helpView() string {
	if m.ctrlCPressed && m.state == stateIdle && m.textArea.Value() == "" {
		return helpStyle.Render("Press Ctrl+C again to quit.")
	}

	switch m.state {
	case stateThinking, stateGenerating:
		statusText := generatingHelpStyle.Render(" Generating... • Ctrl+U/D to scroll • Ctrl+C to cancel")
		return fmt.Sprintf("%s%s", m.spinner.View(), statusText)
	case stateCancelling:
		statusText := generatingHelpStyle.Render(" Cancelling...")
		return fmt.Sprintf("%s%s", m.spinner.View(), statusText)
	}

	help := "Ctrl+C to clear/quit"
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

	var palette string
	if m.showPalette {
		palette = m.paletteView() + "\n"
	}

	return fmt.Sprintf("%s\n%s%s\n%s",
		m.viewport.View(),
		palette,
		textAreaStyle.Render(m.textArea.View()),
		m.helpView(),
	)
}
