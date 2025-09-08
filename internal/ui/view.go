package ui

import (
	"coder/internal/core"
	"fmt"
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
			var block string
			if strings.HasPrefix(msg.Content, "/") {
				blockWidth := m.viewport.Width - commandInputStyle.GetHorizontalPadding()
				block = commandInputStyle.Width(blockWidth).Render(msg.Content)
			} else {
				blockWidth := m.viewport.Width - userInputStyle.GetHorizontalPadding()
				block = userInputStyle.Width(blockWidth).Render(msg.Content)
			}
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
		case core.CommandResultMessage:
			blockWidth := m.viewport.Width - cmdResultStyle.GetHorizontalPadding()
			cmdResultBlock := cmdResultStyle.Width(blockWidth).Render(msg.Content)
			parts = append(parts, cmdResultBlock)
		case core.CommandErrorResultMessage:
			blockWidth := m.viewport.Width - cmdErrorStyle.GetHorizontalPadding()
			cmdErrorBlock := cmdErrorStyle.Width(blockWidth).Render(msg.Content)
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

	return helpStyle.Render("Ctrl+J to send • Enter for newline (or /cmd) • Ctrl+U/D to scroll • Ctrl+C to clear/quit")
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
