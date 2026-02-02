package ui

import (
	"strings"
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var b strings.Builder
	if m.State == stateHistorySelect {
		b.WriteString(m.historyHeaderView())
	}
	b.WriteString(m.Chat.Viewport.View())
	b.WriteString("\n")

	if m.State != stateHistorySelect {
		b.WriteString(textAreaStyle.Render(m.Chat.TextArea.View()))
		b.WriteString("\n")
	}
	b.WriteString(m.StatusView())

	return b.String()
}
