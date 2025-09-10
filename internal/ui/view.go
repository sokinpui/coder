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
	modelInfo := fmt.Sprintf("Model: %s", m.generator.Config.ModelCode)

	helpPart := helpStyle.Render(help)
	modelPart := helpStyle.Render(modelInfo)

	spacing := m.width - lipgloss.Width(helpPart) - lipgloss.Width(modelPart)
	if spacing < 1 {
		return helpPart
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, helpPart, strings.Repeat(" ", spacing), modelPart)

}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	return fmt.Sprintf("%s\n%s\n%s",
		m.viewport.View(),
		textAreaStyle.Render(m.textArea.View()),
		m.helpView(),
	)
}
