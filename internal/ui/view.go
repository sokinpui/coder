package ui

import (
	"fmt"
	"strings"
)

// renderConversation renders the entire message history.
func (m Model) renderConversation() string {
	var parts []string
	for _, msg := range m.messages {
		switch msg.mType {
		case userMessage:
			blockWidth := m.viewport.Width - userInputStyle.GetHorizontalPadding()
			userInputBlock := userInputStyle.Width(blockWidth).Render(msg.content)
			parts = append(parts, userInputBlock)
		case aiMessage:
			var content string
			if msg.content != "" {
				renderedAI, err := m.glamourRenderer.Render(msg.content)
				if err != nil {
					renderedAI = msg.content
				}
				content = renderedAI
			}
			parts = append(parts, content)
		case commandResultMessage:
			blockWidth := m.viewport.Width - cmdResultStyle.GetHorizontalPadding()
			cmdResultBlock := cmdResultStyle.Width(blockWidth).Render(msg.content)
			parts = append(parts, cmdResultBlock)
		}
	}
	return strings.Join(parts, "\n")
}

// helpView renders the help text.
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

// View renders the program's UI.
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
