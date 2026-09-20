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
	case overlaySelector:
		return m.handleKeyPressSelector(msg)
	}

	keyStr := msg.String()
	km := m.Session.GetConfig().Keymap

	// Handle global keybindings first
	switch keyStr {
	case km.ContextList:
		newModel, cmd := m.showQuickView("/list")
		return newModel, cmd, true
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
	case stateAsking, stateThinking, stateGenerating:
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

func (m Model) handleKeyPressSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.Selector.KeyHandler != nil {
		if newMod, cmd, handled := m.Selector.KeyHandler(m, msg); handled {
			return newMod, cmd, true
		}
	}

	prevGGPressed := m.Selector.GGPressed
	m.Selector.GGPressed = false

	if m.Selector.IsSearching {
		switch msg.Type {
		case tea.KeyUp, tea.KeyCtrlK, tea.KeyCtrlP:
			m.moveSelectorCursor(-1)
			return m, nil, true
		case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
			m.moveSelectorCursor(1)
			return m, nil, true
		case tea.KeyEnter:
			if m.Selector.IsLoading {
				return m, nil, true
			}
			if len(m.Selector.Tabs) > 0 {
				m.Selector.IsSearching = false
				m.Selector.SearchInput.Blur()
				return m, nil, true
			}
			return m.confirmSelector()
		case tea.KeyEsc, tea.KeyCtrlC:
			if len(m.Selector.Tabs) > 0 {
				m.Selector.IsSearching = false
				m.Selector.SearchInput.Blur()
				m.Selector.SearchInput.Reset()
				m.Selector.UpdateFilter()
				return m, nil, true
			}
			return m.cancelSelector()
		case tea.KeyTab:
			if m.Selector.IsLoading {
				return m, nil, true
			}
			return m.toggleSelectorItem(1)
		case tea.KeyShiftTab:
			if m.Selector.IsLoading {
				return m, nil, true
			}
			return m.toggleSelectorItem(-1)
		}

		var cmd tea.Cmd
		m.Selector.SearchInput, cmd = m.Selector.SearchInput.Update(msg)
		m.Selector.UpdateFilter()
		return m, cmd, true
	}

	switch msg.Type {
	case tea.KeyTab, tea.KeyShiftTab:
		if m.Selector.IsLoading {
			return m, nil, true
		}
		if len(m.Selector.Tabs) > 0 {
			dir := 1
			if msg.Type == tea.KeyShiftTab {
				dir = len(m.Selector.Tabs) - 1
			}
			nextTab := (m.Selector.ActiveTab + dir) % len(m.Selector.Tabs)
			if m.Selector.OnTabChange != nil {
				newMod, cmd := m.Selector.OnTabChange(m, nextTab)
				return newMod, cmd, true
			}
			m.Selector.ActiveTab = nextTab
			return m, nil, true
		}
		dir := 1
		if msg.Type == tea.KeyShiftTab {
			dir = -1
		}
		return m.toggleSelectorItem(dir)

	case tea.KeyUp, tea.KeyCtrlK:
		m.moveSelectorCursor(-1)
		return m, nil, true

	case tea.KeyDown, tea.KeyCtrlJ:
		m.moveSelectorCursor(1)
		return m, nil, true

	case tea.KeyEsc, tea.KeyCtrlC:
		return m.cancelSelector()

	case tea.KeyEnter:
		if m.Selector.IsLoading {
			return m, nil, true
		}
		return m.confirmSelector()

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q":
			return m.cancelSelector()
		case "h":
			if len(m.Selector.Tabs) > 0 && m.Selector.ActiveTab != 0 {
				if m.Selector.OnTabChange != nil {
					newMod, cmd := m.Selector.OnTabChange(m, 0)
					return newMod, cmd, true
				}
				m.Selector.ActiveTab = 0
				return m, nil, true
			}
		case "l":
			if len(m.Selector.Tabs) > 0 && m.Selector.ActiveTab != 1 {
				if m.Selector.OnTabChange != nil {
					newMod, cmd := m.Selector.OnTabChange(m, 1)
					return newMod, cmd, true
				}
				m.Selector.ActiveTab = 1
				return m, nil, true
			}
		case "/":
			if m.Selector.ShowSearch {
				m.Selector.IsSearching = true
				m.Selector.SearchInput.Focus()
				m.Selector.SearchInput.Reset()
				m.Selector.UpdateFilter()
				return m, nil, true
			}
		case "g":
			if prevGGPressed {
				m.Selector.Cursor = 0
				m = m.notifySelectorCursorChange()
			} else {
				m.Selector.GGPressed = true
			}
			return m, nil, true
		case "G":
			if len(m.Selector.FilteredItems) > 0 {
				m.Selector.Cursor = len(m.Selector.FilteredItems) - 1
				m = m.notifySelectorCursorChange()
			}
			return m, nil, true
		case "u":
			m.scrollSelectorHalfPage(false)
			return m, nil, true
		case "d":
			m.scrollSelectorHalfPage(true)
			return m, nil, true
		case "j":
			m.moveSelectorCursor(1)
			return m, nil, true
		case "k":
			m.moveSelectorCursor(-1)
			return m, nil, true
		}
	}

	return m, nil, true
}

func (m Model) confirmSelector() (tea.Model, tea.Cmd, bool) {
	if m.Selector.OnConfirm != nil {
		selected := m.Selector.GetSelectedItems()
		primary := m.Selector.GetPrimaryItem()
		newModel, cmd := m.Selector.OnConfirm(m, selected, primary)
		return newModel, cmd, true
	}
	m.ActiveOverlay = overlayNone
	return m, nil, true
}

func (m Model) cancelSelector() (tea.Model, tea.Cmd, bool) {
	if m.Selector.OnCancel != nil {
		newModel, cmd := m.Selector.OnCancel(m)
		return newModel, cmd, true
	}
	m.ActiveOverlay = overlayNone
	if m.State == stateIdle {
		m.Chat.TextArea.Focus()
		return m, textarea.Blink, true
	}
	return m, nil, true
}

func (m *Model) moveSelectorCursor(delta int) {
	total := len(m.Selector.FilteredItems)
	if total == 0 {
		return
	}
	newCursor := m.Selector.Cursor + delta
	if newCursor < 0 {
		newCursor = 0
	} else if newCursor >= total {
		newCursor = total - 1
	}
	m.Selector.Cursor = newCursor
	*m = m.notifySelectorCursorChange()
}

func (m *Model) scrollSelectorHalfPage(down bool) {
	total := len(m.Selector.FilteredItems)
	if total == 0 {
		return
	}
	scrollAmount := max(1, m.Selector.Height/2)
	if down {
		m.Selector.Cursor = min(total-1, m.Selector.Cursor+scrollAmount)
	} else {
		m.Selector.Cursor = max(0, m.Selector.Cursor-scrollAmount)
	}
	*m = m.notifySelectorCursorChange()
}

func (m Model) notifySelectorCursorChange() Model {
	if m.Selector.OnCursorChange != nil {
		primary := m.Selector.GetPrimaryItem()
		return m.Selector.OnCursorChange(m, primary)
	}
	return m
}

func (m Model) toggleSelectorItem(dir int) (tea.Model, tea.Cmd, bool) {
	if len(m.Selector.FilteredItems) == 0 || m.Selector.Cursor >= len(m.Selector.FilteredItems) {
		return m, nil, true
	}
	item := m.Selector.FilteredItems[m.Selector.Cursor]
	if _, exists := m.Selector.Selected[item.ID]; exists {
		delete(m.Selector.Selected, item.ID)
	} else {
		m.Selector.Selected[item.ID] = struct{}{}
	}
	m.moveSelectorCursor(dir)
	return m, nil, true
}
