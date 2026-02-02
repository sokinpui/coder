package ui

import (
	"github.com/sokinpui/coder/internal/history"

	"github.com/sahilm/fuzzy"
	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPressHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.IsHistorySearching {
		switch msg.Type {
		case tea.KeyEnter:
			m.IsHistorySearching = false
			m.HistorySearchInput.Blur()
			m.Viewport.SetContent(m.historyView())
			return m, nil, true
		case tea.KeyEsc, tea.KeyCtrlC:
			m.IsHistorySearching = false
			m.HistorySearchInput.Blur()
			m.HistorySearchInput.Reset()
			m.updateHistoryFilter()
			m.Viewport.SetContent(m.historyView())
			return m, nil, true
		}

		var cmd tea.Cmd
		m.HistorySearchInput, cmd = m.HistorySearchInput.Update(msg)
		m.updateHistoryFilter()
		m.Viewport.SetContent(m.historyView())
		return m, cmd, true
	}

	prevGGPressed := m.HistoryGGPressed
	m.HistoryGGPressed = false // Reset by default

	switch msg.Type {
	case tea.KeyCtrlD:
		if len(m.FilteredHistoryItems) == 0 {
			return m, nil, true
		}
		scrollAmount := m.Viewport.Height / 2
		m.Viewport.HalfPageDown()
		m.HistoryCussorPos = cursorPosAfterScroll(m.HistoryCussorPos, scrollAmount, len(m.FilteredHistoryItems), true)
		m.Viewport.SetContent(m.historyView())
		return m, nil, true
	case tea.KeyCtrlU:
		if len(m.FilteredHistoryItems) == 0 {
			return m, nil, true
		}
		scrollAmount := m.Viewport.Height / 2
		m.Viewport.HalfPageUp()
		m.HistoryCussorPos = cursorPosAfterScroll(m.HistoryCussorPos, scrollAmount, len(m.FilteredHistoryItems), false)
		m.Viewport.SetContent(m.historyView())
		return m, nil, true

	case tea.KeyUp, tea.KeyCtrlK:
		m.moveHistoryCursor(-1)
		return m, nil, true

	case tea.KeyDown, tea.KeyCtrlJ:
		m.moveHistoryCursor(1)
		return m, nil, true

	case tea.KeyEsc, tea.KeyCtrlC:
		m.HistoryItems = nil
		if m.IsStreaming {
			// Return to the generation view
			messages := m.Session.GetMessages()
			if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
				m.State = stateThinking
			} else {
				m.State = stateGenerating
			}
			delay := m.Session.GetConfig().Generation.StreamDelay
			m.Viewport.SetContent(m.renderConversation())
			// Re-issue commands needed for generation state
			return m, tea.Batch(listenForStream(m.StreamSub), streamAnimeCmd(delay), m.Spinner.Tick), true
		} else {
			// Return to idle
			m.State = stateIdle
			m.TextArea.Focus()
			m.Viewport.SetContent(m.renderConversation())
			return m, textarea.Blink, true
		}

	case tea.KeyEnter:
		if len(m.FilteredHistoryItems) == 0 || m.HistoryCussorPos >= len(m.FilteredHistoryItems) {
			return m, nil, true
		}
		selectedItem := m.FilteredHistoryItems[m.HistoryCussorPos]
		if m.IsStreaming {
			m.Session.CancelGeneration()
			m.IsStreaming = false // Prevent streamFinishedMsg from running
			m.StreamSub = nil
		}
		return m, loadConversationCmd(m.Session, selectedItem.Filename), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "/":
			m.IsHistorySearching = true
			m.HistorySearchInput.Focus()
			m.HistorySearchInput.Reset()
			m.updateHistoryFilter()
			m.Viewport.SetContent(m.historyView())
			return m, nil, true
		case "g":
			if prevGGPressed {
				m.HistoryCussorPos = 0
				m.Viewport.GotoTop()
				m.Viewport.SetContent(m.historyView())
			} else {
				m.HistoryGGPressed = true
			}
			return m, nil, true
		case "G":
			if len(m.FilteredHistoryItems) > 0 {
				m.HistoryCussorPos = len(m.FilteredHistoryItems) - 1
				m.Viewport.GotoBottom()
				m.Viewport.SetContent(m.historyView())
			}
			return m, nil, true
		case "j":
			m.moveHistoryCursor(1)
			return m, nil, true
		case "k":
			m.moveHistoryCursor(-1)
			return m, nil, true
		}
	}
	return m, nil, true
}

func (m *Model) moveHistoryCursor(delta int) {
	newPos := m.HistoryCussorPos + delta
	if newPos < 0 || newPos >= len(m.FilteredHistoryItems) {
		return
	}

	m.HistoryCussorPos = newPos

	// Viewport auto-scroll logic
	const headerHeight = 2
	cursorLine := m.HistoryCussorPos + headerHeight

	// Scroll Up
	if delta < 0 {
		if cursorLine < m.Viewport.YOffset {
			m.Viewport.LineUp(1)
		}
	}

	// Scroll Down
	if delta > 0 {
		// We use a buffer of 1 line to keep the selection clearly visible
		if cursorLine >= m.Viewport.YOffset+m.Viewport.Height-1 {
			m.Viewport.LineDown(1)
		}
	}

	m.Viewport.SetContent(m.historyView())
}

func (m *Model) updateHistoryFilter() {
	query := m.HistorySearchInput.Value()
	if query == "" {
		m.FilteredHistoryItems = m.HistoryItems
		return
	}

	targets := make([]string, len(m.HistoryItems))
	for i, item := range m.HistoryItems {
		targets[i] = item.Title + " " + item.Filename
	}

	matches := fuzzy.Find(query, targets)
	var filtered []history.ConversationInfo
	for _, match := range matches {
		filtered = append(filtered, m.HistoryItems[match.Index])
	}

	m.FilteredHistoryItems = filtered
	if m.HistoryCussorPos >= len(m.FilteredHistoryItems) {
		m.HistoryCussorPos = max(0, len(m.FilteredHistoryItems)-1)
	}
}
