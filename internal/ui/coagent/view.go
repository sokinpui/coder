package coagentui

import (
	"fmt"
)

func (m Model) View() string {
	if m.state == stateRunning {
		status := m.statusText
		if status == "" {
			status = "Agent working..."
		}
		spinnerPart := m.spinner.View()
		statusPart := spinnerStyle.Render(status)
		helpPart := toolMutedStyle.Render("(Ctrl+C to interrupt)")
		return fmt.Sprintf("%s %s %s", spinnerPart, statusPart, helpPart)
	}

	return promptPrefixStyle.Render("❯ ") + m.input.View()
}
