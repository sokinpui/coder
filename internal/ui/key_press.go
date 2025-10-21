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

	// Handle global keybindings first
	switch msg.Type {
	case tea.KeyCtrlZ:
		return m, tea.Suspend, true // Suspend the application
	case tea.KeyCtrlQ:
		if m.State == stateQuickView {
			m.State = m.PreviousState
			m.QuickView.Visible = false
			if m.State == stateIdle {
				m.TextArea.Focus()
				return m, textarea.Blink, true
			}
			var cmd tea.Cmd
			isGeneratingState := m.State == stateGenPending || m.State == stateThinking || m.State == stateGenerating || m.State == stateCancelling
			if isGeneratingState {
				cmd = m.Spinner.Tick
			}
			return m, cmd, true
		}
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
		m.PreviousState = m.State
		m.State = stateQuickView
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
	case stateQuickView:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ:
			m.State = m.PreviousState
			m.QuickView.Visible = false
			if m.State == stateIdle {
				m.TextArea.Focus()
				return m, textarea.Blink, true
			}
			var cmd tea.Cmd
			isGeneratingState := m.State == stateGenPending || m.State == stateThinking || m.State == stateGenerating || m.State == stateCancelling
			if isGeneratingState {
				cmd = m.Spinner.Tick
			}
			return m, cmd, true
		}
		cmd := m.QuickView.Update(msg)
		return m, cmd, true
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
