package coagentui

import "strings"

func (m Model) View() string {
	if m.state == stateRunning {
		var sb strings.Builder
		sb.WriteString(m.spinner.View())
		sb.WriteString(" ")
		sb.WriteString(spinnerStyle.Render("Agent working... (Ctrl+C to interrupt)"))
		if m.partialLine != "" {
			sb.WriteString("\n")
			sb.WriteString(m.partialLine)
		}
		return sb.String()
	}

	return promptPrefixStyle.Render("❯ ") + m.input.View()
}
