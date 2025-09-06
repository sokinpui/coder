package ui

import (
	"fmt"
	"strings"
)

// renderConversation renders the entire message history.
func (m Model) renderConversation() string {
	var parts []string
	for _, msg := range m.messages {
		if msg.isUser {
			blockWidth := m.viewport.Width - userInputStyle.GetHorizontalPadding()
			userInputBlock := userInputStyle.Width(blockWidth).Render(msg.content)
			parts = append(parts, userInputBlock)
		} else {
			var content string
			if msg.content != "" {
				renderedAI, err := m.glamourRenderer.Render(msg.content)
				if err != nil {
					renderedAI = msg.content
				}
				content = renderedAI
			}
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n")
}

// helpView renders the help text.
func (m Model) helpView() string {
	if m.ctrlCPressed && m.state == stateIdle && m.textArea.Value() == "" {
		return helpStyle.Render("Press Ctrl+C again to quit.")
	}

	if m.state == stateCancelling {
		return generatingHelpStyle.Render("Cancelling...")
	}

	return ""
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
