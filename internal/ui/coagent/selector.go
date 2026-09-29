package coagentui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/token"
	coderui "github.com/sokinpui/coder/internal/ui/coder"
)

func (m Model) openHistorySelector(initialTab int) (tea.Model, tea.Cmd) {
	m.showSelector = true
	m.activeOverlay = overlaySelector
	m.selector = coderui.NewSelector()
	m.selector.Tabs = []string{"History", "Active"}
	m.selector.ActiveTab = initialTab
	m.selector.ShowSearch = true
	m.selector.IsSearching = false
	m.selector.FooterHelp = "── [Esc/q: close | /: search | Tab/h/l: switch tab | Enter: load] ──"

	if initialTab == 1 {
		m = m.refreshActiveSelectorItems()
		return m, nil
	}

	return m, func() tea.Msg {
		items, err := m.session.HistoryManager.ListConversationsByMode(coagent.ModeCoAgent)
		return historyListResultMsg{items: items, err: err}
	}
}

func (m Model) refreshActiveSelectorItems() Model {
	if m.selector.ActiveTab != 1 {
		return m
	}
	var items []coderui.SelectorItem
	for i := len(m.activeSessions) - 1; i >= 0; i-- {
		sess := m.activeSessions[i]
		marker := ""
		if sess.ID == m.session.ID {
			marker = "*"
		}
		items = append(items, coderui.SelectorItem{
			ID:          sess.ID,
			Title:       sess.Title,
			Description: marker,
			Data:        sess,
		})
	}
	m.selector.SetItems(items)
	return m
}

func (m Model) newSession() (tea.Model, tea.Cmd) {
	sess, err := coagent.NewSession(m.cfg)
	if err != nil {
		return m, nil
	}
	m.addActiveSession(sess)
	m.session = sess
	m.tokenCount = token.CountTokens(sess.GetPrompt())
	m.isAIRendering = false
	m.pendingAIRender = false
	m.viewport.ClearCache()
	m.viewport.GotoTop()
	return m.updateViewportContent(), nil
}

func (m Model) handleSelectorUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}

	if !m.selector.ShowSearch && len(m.selector.Tabs) == 0 {
			switch keyMsg.Type {
			case tea.KeyEsc, tea.KeyCtrlC:
				m.showSelector = false
				m.activeOverlay = overlayNone
				m.input.Model.Focus()
				return m, nil
			}
			return m.handleAtomicMsgKey(keyMsg)
		}

	if m.selector.IsSearching {
		return m.handleSelectorSearchKey(keyMsg)
	}
	return m.handleSelectorNavKey(keyMsg)
}

func (m Model) handleSelectorSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyCtrlK, tea.KeyCtrlP:
		m.moveSelectorCursor(-1)
		return m, nil
	case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
		m.moveSelectorCursor(1)
		return m, nil
	case tea.KeyEnter:
		m.selector.IsSearching = false
		m.selector.SearchInput.Blur()
		return m, nil
	case tea.KeyEsc, tea.KeyCtrlC:
		m.selector.IsSearching = false
		m.selector.SearchInput.Blur()
		m.selector.SearchInput.Reset()
		m.selector.UpdateFilter()
		return m, nil
	}
	var cmd tea.Cmd
	m.selector.SearchInput, cmd = m.selector.SearchInput.Update(msg)
	m.selector.UpdateFilter()
	return m, cmd
}

func (m Model) handleSelectorNavKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prevGGPressed := m.selector.GGPressed
	m.selector.GGPressed = false

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.showSelector = false
		m.activeOverlay = overlayNone
		m.input.Model.Focus()
		return m, nil
	case tea.KeyEnter:
		return m.confirmSelector()
	case tea.KeyUp, tea.KeyCtrlK, tea.KeyCtrlP:
		m.moveSelectorCursor(-1)
		return m, nil
	case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
		m.moveSelectorCursor(1)
		return m, nil
	case tea.KeyTab, tea.KeyShiftTab:
			dir := 1
			if msg.Type == tea.KeyShiftTab {
				dir = -1
			}
			m.selector.ActiveTab = (m.selector.ActiveTab + dir + len(m.selector.Tabs)) % len(m.selector.Tabs)
			m.selector.Cursor = 0
			m.selector.SearchInput.Reset()
			if m.selector.ActiveTab == 1 {
				m = m.refreshActiveSelectorItems()
				return m, nil
			}
			return m, func() tea.Msg {
				items, err := m.session.HistoryManager.ListConversationsByMode(coagent.ModeCoAgent)
				return historyListResultMsg{items: items, err: err}
			}
		case tea.KeyRunes:
			switch string(msg.Runes) {
		case "h":
			if m.selector.ActiveTab != 0 {
				m.selector.ActiveTab = 0
				m.selector.Cursor = 0
				m.selector.SearchInput.Reset()
				return m, func() tea.Msg {
					items, err := m.session.HistoryManager.ListConversationsByMode(coagent.ModeCoAgent)
					return historyListResultMsg{items: items, err: err}
				}
			}
		case "l":
			if m.selector.ActiveTab != 1 {
				m.selector.ActiveTab = 1
				m.selector.Cursor = 0
				m.selector.SearchInput.Reset()
				m = m.refreshActiveSelectorItems()
				return m, nil
			}
		case "q":
			m.showSelector = false
			m.activeOverlay = overlayNone
			m.input.Model.Focus()
			return m, nil
		case "/":
			if m.selector.ShowSearch {
				m.selector.IsSearching = true
				m.selector.SearchInput.Focus()
				m.selector.SearchInput.Reset()
				m.selector.UpdateFilter()
				return m, nil
			}
		case "j":
			m.moveSelectorCursor(1)
			return m, nil
		case "k":
			m.moveSelectorCursor(-1)
			return m, nil
		case "u":
			m.scrollSelectorHalfPage(false)
			return m, nil
		case "d":
			m.scrollSelectorHalfPage(true)
			return m, nil
		case "g":
			if prevGGPressed {
				m.selector.Cursor = 0
			} else {
				m.selector.GGPressed = true
			}
			return m, nil
		case "G":
			total := len(m.selector.FilteredItems)
			if total > 0 {
				m.selector.Cursor = total - 1
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) moveSelectorCursor(delta int) {
	total := len(m.selector.FilteredItems)
	if total > 0 {
		m.selector.Cursor = max(0, min(total-1, m.selector.Cursor+delta))
	}
}

func (m *Model) scrollSelectorHalfPage(down bool) {
	total := len(m.selector.FilteredItems)
	if total == 0 {
		return
	}
	scrollAmount := max(1, m.selector.Height/2)
	if down {
		m.selector.Cursor = min(total-1, m.selector.Cursor+scrollAmount)
	} else {
		m.selector.Cursor = max(0, m.selector.Cursor-scrollAmount)
	}
}

func (m Model) confirmSelector() (tea.Model, tea.Cmd) {
	item := m.selector.GetPrimaryItem()
	if item == nil {
		m.showSelector = false
		m.activeOverlay = overlayNone
		m.input.Model.Focus()
		return m, nil
	}

	if m.selector.ActiveTab == 1 {
		if target := m.getSessionByID(item.ID); target != nil {
			m.session = target
			m.tokenCount = token.CountTokens(m.session.GetPrompt())
			m.showSelector = false
			m.activeOverlay = overlayNone
			m.isAIRendering = false
			m.pendingAIRender = false
			m.input.Model.Focus()
			m.viewport.ClearCache()
			m.viewport.GotoBottom()
			return m.updateViewportContent(), nil
		}
	}
	for _, s := range m.activeSessions {
		if s.HistoryFilename == item.ID {
			m.session = s
			m.tokenCount = token.CountTokens(m.session.GetPrompt())
			m.showSelector = false
			m.activeOverlay = overlayNone
			m.isAIRendering = false
			m.pendingAIRender = false
			m.input.Model.Focus()
			m.viewport.ClearCache()
			m.viewport.GotoBottom()
			return m.updateViewportContent(), nil
		}
	}
	_ = m.session.LoadConversation(item.ID)
	m.addActiveSession(m.session)
	m.tokenCount = token.CountTokens(m.session.GetPrompt())
	m.showSelector = false
	m.activeOverlay = overlayNone
	m.input.Model.Focus()
	m.viewport.ClearCache()
	m.viewport.GotoBottom()
	return m.updateViewportContent(), nil
}
