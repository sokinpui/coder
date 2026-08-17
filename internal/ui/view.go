package ui

import (
	"strings"
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.Chat.Viewport.View())
	b.WriteString("\n")
	b.WriteString(m.inputView())
	b.WriteString("\n")
	b.WriteString(m.StatusView())

	return b.String()
}

func (m Model) inputView() string {
	return textAreaStyle.Render(m.Chat.TextArea.View())
}
