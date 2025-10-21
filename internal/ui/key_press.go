package ui

import (
	"coder/internal/types"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type != tea.KeyCtrlC {
		m.CtrlCPressed = false
	}

	// If quick view is visible, it gets priority on key presses.
	if m.QuickView.Visible {
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ:
			m.QuickView.Visible = false
			if m.State == stateIdle {
				m.TextArea.Focus()
				return m, textarea.Blink, true
			}
			return m, nil, true
		}
		// Let quickview handle its own scrolling etc.
		cmd := m.QuickView.Update(msg)
		return m, cmd, true
	}

	// Handle global keybindings first
	switch msg.Type {
	case tea.KeyCtrlZ:
		return m, tea.Suspend, true // Suspend the application
	case tea.KeyCtrlQ:
		// This now just opens the quick view
		messages := m.Session.GetMessages()
		var lastTwo []types.Message
		count := 0
		for i := len(messages) - 1; i >= 0 && count < 2; i-- {
			msg := messages[i]
			switch msg.Type {
			case types.UserMessage, types.AIMessage:
				if msg.Content != "" { // Don't show empty AI messages
					lastTwo = append(lastTwo, msg)
					count++
				}
			}
		}
		// reverse to get correct order
		for i, j := 0, len(lastTwo)-1; i < j; i, j = i+1, j-1 {
			lastTwo[i], lastTwo[j] = lastTwo[j], lastTwo[i]
		}

		m.QuickView.SetMessages(lastTwo)
		m.QuickView.Visible = true
		m.TextArea.Blur()
		return m, nil, true
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
