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
	case overlayPicker:
		return m.handleKeyPressPicker(msg)
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

func (m Model) handleKeyPressPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.ActiveOverlay = overlayNone
		m.Picker.TextInput.Blur()
		m.Picker.TextInput.Reset()
		m.Picker.Selected = make(map[string]struct{})
		m.Picker.OnSelect = nil
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true
	case tea.KeyTab:
		if len(m.Picker.FoundItems) == 0 || m.Picker.Cursor >= len(m.Picker.FoundItems) {
			return m, nil, true
		}
		item := m.Picker.FoundItems[m.Picker.Cursor]
		if _, exists := m.Picker.Selected[item]; exists {
			delete(m.Picker.Selected, item)
		} else {
			m.Picker.Selected[item] = struct{}{}
		}
		if m.Picker.Cursor < len(m.Picker.FoundItems)-1 {
			m.Picker.Cursor++
		}
		return m, nil, true
	case tea.KeyShiftTab:
		if len(m.Picker.FoundItems) == 0 || m.Picker.Cursor >= len(m.Picker.FoundItems) {
			return m, nil, true
		}
		item := m.Picker.FoundItems[m.Picker.Cursor]
		if _, exists := m.Picker.Selected[item]; exists {
			delete(m.Picker.Selected, item)
		} else {
			m.Picker.Selected[item] = struct{}{}
		}
		if m.Picker.Cursor > 0 {
			m.Picker.Cursor--
		}
		return m, nil, true
	case tea.KeyUp, tea.KeyCtrlP, tea.KeyCtrlK:
		if m.Picker.Cursor > 0 {
			m.Picker.Cursor--
		}
		return m, nil, true
	case tea.KeyDown, tea.KeyCtrlN, tea.KeyCtrlJ:
		if m.Picker.Cursor < len(m.Picker.FoundItems)-1 {
			m.Picker.Cursor++
		}
		return m, nil, true
	case tea.KeyEnter:
		selectedList := m.Picker.getSelectedItems()
		primary := m.Picker.getPrimaryItem()
		action := m.Picker.OnSelect

		m.ActiveOverlay = overlayNone
		m.Picker.TextInput.Blur()
		m.Picker.TextInput.Reset()
		m.Picker.Selected = make(map[string]struct{})
		m.Picker.OnSelect = nil

		if action != nil {
			newModel, cmd := action(m, selectedList, primary)
			return newModel, cmd, true
		}
		return m, nil, true
	}

	var cmd tea.Cmd
	m.Picker.TextInput, cmd = m.Picker.TextInput.Update(msg)
	m.Picker.updateFoundItems()
	return m, cmd, true
}
