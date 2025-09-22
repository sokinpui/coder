package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type != tea.KeyCtrlC {
		m.ctrlCPressed = false
	}

	// Handle scrolling regardless of state.
	switch msg.Type {
	case tea.KeyCtrlU:
		m.viewport.HalfViewUp()
		return m, nil, true
	case tea.KeyCtrlD:
		m.viewport.HalfViewDown()
		return m, nil, true
	}

	switch m.state {
	case stateThinking, stateGenerating, stateCancelling:
		return m.handleKeyPressGenerating(msg)
	case stateHistorySelect:
		return m.handleKeyPressHistory(msg)
	case stateVisualSelect:
		return m.handleKeyPressVisual(msg)
	case stateIdle:
		return m.handleKeyPressIdle(msg)
	}
	return m, nil, false
}
