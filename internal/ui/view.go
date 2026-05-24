package ui

import (
	"strings"
	"github.com/charmbracelet/lipgloss"
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
	b.WriteString(m.inputView())
	b.WriteString("\n")
	b.WriteString(m.StatusView())

	return b.String()
}

func (m Model) inputView() string {
	if m.State == stateIdle {
		return textAreaStyle.Render(m.Chat.TextArea.View())
	}

	var statusText string
	switch m.State {
	case stateGenerating, stateThinking, stateCancelling, stateGenPending:
		statusText = "AI is thinking..."
	case stateVisualSelect:
		statusText = "Visual Mode - Use j/k to navigate"
	case stateHistorySelect:
		statusText = "History Mode - Select a conversation"
	case stateTree:
		statusText = "Tree Mode - Select files/dirs"
	case stateFinder:
		statusText = "Search Mode..."
	default:
		statusText = "Input disabled"
	}

	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		Width(m.Chat.TextArea.Width()).
		Height(m.Chat.TextArea.Height()).
		Padding(0, 1).
		Render(statusText)

	return textAreaStyle.Render(content)
}
