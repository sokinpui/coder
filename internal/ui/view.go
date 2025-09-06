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
	help := helpStyle.Render("Ctrl+J to submit, Ctrl+C to quit")
	switch m.state {
	case stateThinking:
		help = generatingHelpStyle.Render("Thinking... Ctrl+C to quit")
	case stateGenerating:
		help = generatingHelpStyle.Render("Generating... Ctrl+C to quit")
	}

	return help
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
