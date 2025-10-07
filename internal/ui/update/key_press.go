package update

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type != tea.KeyCtrlC {
		m.CtrlCPressed = false
	}

	// Handle scrolling for non-history states.
	if m.State != stateHistorySelect {
		switch msg.Type {
		case tea.KeyCtrlU:
			m.Viewport.HalfViewUp()
			return m, nil, true
		case tea.KeyCtrlD:
			m.Viewport.HalfViewDown()
			return m, nil, true
		}
	}

	switch m.State {
	case stateGenPending:
		return m.handleKeyPressGenPending(msg)
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
