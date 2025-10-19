package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type != tea.KeyCtrlC {
		m.CtrlCPressed = false
	}

	// Handle global keybindings first
	switch msg.Type {
	case tea.KeyCtrlZ:
		return m, tea.Suspend, true // Suspend the application
	}

	// Handle scrolling for non-history states.
	if m.State != stateHistorySelect {
		switch msg.Type {
		case tea.KeyCtrlU:
			m.Viewport.HalfPageUp()
			return m, nil, true
		case tea.KeyCtrlD:
			m.Viewport.HalfPageDown()
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
	case stateFinder:
		newFinder, cmd := m.Finder.Update(msg)
		m.Finder = newFinder
		if !m.Finder.Visible {
			m.State = stateIdle
			m.TextArea.Focus()
			return m, tea.Batch(cmd, textinput.Blink), true
		}
		return m, cmd, true
	case stateIdle:
		return m.handleKeyPressIdle(msg)
	}
	return m, nil, false
}
