package ui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.Type != tea.KeyCtrlC {
		m.Chat.CtrlCPressed = false
	}

	switch m.ActiveOverlay {
	case overlayQuickView:
		return m.handleKeyPressQuickView(msg)
	case overlayHistory:
		return m.handleKeyPressHistory(msg)
	case overlayAtomicMsg:
		return m.handleKeyPressAtomicMsg(msg)
	case overlayFinder:
		return m.handleKeyPressFinder(msg)
	}

	keyStr := msg.String()
	km := m.Session.GetConfig().Keymap

	// Handle global keybindings first
	switch keyStr {
	case km.ContextList:
		event := m.Session.HandleShortcut("/list")
		model, cmd := m.handleEvent(event)
		return model, cmd, true
	case km.Suspend:
		return m, tea.Suspend, true
	case km.ScrollUp:
		m.Chat.Viewport.HalfPageUp()
		return m, nil, true
	case km.ScrollDown:
		m.Chat.Viewport.HalfPageDown()
		return m, nil, true
	}

	switch m.State {
	case stateAsking, stateThinking, stateGenerating, stateCancelling:
		return m.handleKeyPressGenerating(msg)
	case stateIdle:
		return m.handleKeyPressIdle(msg)
	}
	return m, nil, false
}

func (m Model) handleKeyPressQuickView(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ, tea.KeyEnter:
		m.ActiveOverlay = overlayNone
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true
	}

	if msg.Type == tea.KeyRunes && msg.String() == "q" {
		m.ActiveOverlay = overlayNone
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true
	}

	cmd := m.QuickView.Update(msg)
	return m, cmd, true
}

func (m Model) handleKeyPressFinder(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.ActiveOverlay = overlayNone
		m.Finder.TextInput.Blur()
		m.Finder.TextInput.Reset()
		m.Finder.Selected = make(map[string]struct{})
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true
	case tea.KeyTab:
		if len(m.Finder.FoundItems) == 0 || m.Finder.Cursor >= len(m.Finder.FoundItems) {
			return m, nil, true
		}
		item := m.Finder.FoundItems[m.Finder.Cursor]
		if _, exists := m.Finder.Selected[item]; exists {
			delete(m.Finder.Selected, item)
		} else {
			m.Finder.Selected[item] = struct{}{}
		}
		if m.Finder.Cursor < len(m.Finder.FoundItems)-1 {
			m.Finder.Cursor++
		}
		return m, nil, true
	case tea.KeyShiftTab:
		if len(m.Finder.FoundItems) == 0 || m.Finder.Cursor >= len(m.Finder.FoundItems) {
			return m, nil, true
		}
		item := m.Finder.FoundItems[m.Finder.Cursor]
		if _, exists := m.Finder.Selected[item]; exists {
			delete(m.Finder.Selected, item)
		} else {
			m.Finder.Selected[item] = struct{}{}
		}
		if m.Finder.Cursor > 0 {
			m.Finder.Cursor--
		}
		return m, nil, true
	case tea.KeyUp, tea.KeyCtrlP, tea.KeyCtrlK:
		if m.Finder.Cursor > 0 {
			m.Finder.Cursor--
		}
		return m, nil, true
	case tea.KeyDown, tea.KeyCtrlN, tea.KeyCtrlJ:
		if m.Finder.Cursor < len(m.Finder.FoundItems)-1 {
			m.Finder.Cursor++
		}
		return m, nil, true
	case tea.KeyEnter:
		var selectedList []string
		for _, item := range m.Finder.AllItems {
			if _, ok := m.Finder.Selected[item]; ok {
				selectedList = append(selectedList, item)
			}
		}

		if len(selectedList) == 0 && len(m.Finder.FoundItems) > 0 && m.Finder.Cursor < len(m.Finder.FoundItems) {
			selectedList = append(selectedList, m.Finder.FoundItems[m.Finder.Cursor])
		}

		if len(m.Finder.FoundItems) > 0 && m.Finder.Cursor < len(m.Finder.FoundItems) {
			selected := m.Finder.FoundItems[m.Finder.Cursor]
			m.ActiveOverlay = overlayNone
			m.Finder.TextInput.Blur()
			m.Finder.TextInput.Reset()
			m.Finder.Selected = make(map[string]struct{})
			return m, func() tea.Msg {
				return finderResultMsg{
					result:  selected,
					results: selectedList,
					mode:    m.Finder.Mode,
				}
			}, true
		}
		return m, nil, true
	}

	var cmd tea.Cmd
	m.Finder.TextInput, cmd = m.Finder.TextInput.Update(msg)
	m.Finder.updateFoundItems()
	return m, cmd, true
}
