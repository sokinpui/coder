package update

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type != tea.KeyCtrlC {
		m.CtrlCPressed = false
	}

	// Handle scrolling for non-history states.
	if m.State != StateHistorySelect {
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
	case StateGenPending:
		return m.handleKeyPressGenPending(msg)
	case StateThinking, StateGenerating, StateCancelling:
		return m.handleKeyPressGenerating(msg)
	case StateHistorySelect:
		return m.handleKeyPressHistory(msg)
	case StateVisualSelect:
		return m.handleKeyPressVisual(msg)
	case StateIdle:
		return m.handleKeyPressIdle(msg)
	}
	return m, nil, false
}
