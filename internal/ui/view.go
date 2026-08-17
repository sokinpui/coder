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
	if m.State != stateHistorySelect {
		b.WriteString("\n")
		b.WriteString(m.inputView())
	}
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
	case stateAsking:
		statusText = "Asking..."
	case stateThinking:
		statusText = "Thinking..."
	case stateGenerating:
		statusText = "Generating..."
	case stateCancelling:
		statusText = "Cancelling..."
	case stateAtomicMsg:
		statusText = "Atomic Messages - j/k: navigate | v: select | y/d: copy/del | a/e/r/b: actions | Esc/C-c: exit"
	case stateHistorySelect:
		statusText = "History Mode - Select a conversation"
	case stateFinder:
		statusText = "Search Mode..."
	default:
		statusText = "Input disabled"
	}

	// Ensure placeholder area has same dimensions and style as active textarea
	content := disabledPlaceholderStyle.
		Width(m.Chat.TextArea.Width()).
		Height(m.Chat.TextArea.Height()).
		Render(statusText)

	return textAreaStyle.Render(content)
}
