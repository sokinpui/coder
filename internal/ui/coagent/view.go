package coagentui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	inputLine := promptPrefixStyle.Render("❯ ") + m.input.View()
	inputBox := inputContainerStyle.Width(max(10, m.width-2)).Render(inputLine)
	statusBar := m.statusView()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		inputBox,
		statusBar,
	)
}
