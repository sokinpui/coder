package ui

import (
	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type != tea.KeyCtrlC {
		m.Chat.CtrlCPressed = false
	}

	// If quick view is visible, it gets priority on key presses.
	if m.QuickView.Visible {
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ, tea.KeyEnter:
			m.QuickView.Visible = false
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
				return m, textarea.Blink, true
			}
			return m, nil, true
		}

		if msg.Type == tea.KeyRunes && msg.String() == "q" {
			m.QuickView.Visible = false
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
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
	case tea.KeyCtrlL:
		cmdOutput, _, cmdSuccess := commands.ProcessCommand(":list-all", m.Session)

		var messages []types.Message
		messages = append(messages, types.Message{
			Type:    types.CommandMessage,
			Content: ":list-all",
		})

		if !cmdSuccess {
			messages = append(messages, types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: cmdOutput.Payload,
			})
		} else {
			messages = append(messages, types.Message{
				Type:    types.CommandResultMessage,
				Content: cmdOutput.Payload,
			})
		}

		m.QuickView.SetMessages(messages)
		m.QuickView.Visible = true
		m.Chat.TextArea.Blur()
		return m, nil, true
	case tea.KeyCtrlZ:
		return m, tea.Suspend, true // Suspend the application
	case tea.KeyCtrlQ:
		event := m.Session.HandleShortcut(":jump")
		model, cmd := m.handleEvent(event)
		return model, cmd, true
	}

	// Handle scrolling for non-history states.
	if m.State != stateHistorySelect {
		switch msg.Type {
		case tea.KeyCtrlU:
			m.Chat.Viewport.HalfPageUp()
			return m, nil, true
		case tea.KeyCtrlD:
			m.Chat.Viewport.HalfPageDown()
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
			m.Chat.TextArea.Focus()
			return m, tea.Batch(cmd, textinput.Blink), true
		}
		return m, cmd, true
	case stateSearch:
		newSearch, cmd := m.Search.Update(msg)
		m.Search = newSearch
		if !m.Search.Visible {
			m.State = stateIdle
			m.Chat.TextArea.Focus()
			return m, tea.Batch(cmd, textinput.Blink), true
		}
		return m, cmd, true
	case stateTree:
		newTree, cmd := m.Tree.Update(msg)
		m.Tree = newTree
		if !m.Tree.Visible {
			m.State = stateIdle
			m.Chat.TextArea.Focus()
			return m, tea.Batch(cmd, textarea.Blink), true
		}
		return m, cmd, true
	case stateJump:
		newJump, cmd := m.Jump.Update(msg)
		m.Jump = newJump
		if !m.Jump.Visible {
			m.State = stateIdle
			m.Chat.TextArea.Focus()
			return m, tea.Batch(cmd, textarea.Blink), true
		}
		return m, cmd, true
	case stateInitializing:
		fallthrough
	case stateIdle:
		return m.handleKeyPressIdle(msg)
	}
	return m, nil, false
}
