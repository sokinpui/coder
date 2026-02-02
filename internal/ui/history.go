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
			m.Viewport.SetContent(m.historyListView())
			return m, nil, true
		case tea.KeyEsc, tea.KeyCtrlC:
			m.IsHistorySearching = false
			m.HistorySearchInput.Blur()
			m.HistorySearchInput.Reset()
			m.updateHistoryFilter()
			m.Viewport.SetContent(m.historyListView())
			return m, nil, true
		}

		var cmd tea.Cmd
		m.HistorySearchInput, cmd = m.HistorySearchInput.Update(msg)
		m.updateHistoryFilter()
		m.Viewport.SetContent(m.historyListView())
		return m, cmd, true
	}

	prevGGPressed := m.HistoryGGPressed
	m.HistoryGGPressed = false // Reset by default

	switch msg.Type {
	case tea.KeyCtrlD:
		m.scrollHistoryHalfPage(true)
		return m, nil, true
	case tea.KeyCtrlU:
		m.scrollHistoryHalfPage(false)
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
			m.Viewport.SetContent(m.historyListView())
			return m, nil, true
		case "g":
			if prevGGPressed {
				m.HistoryCussorPos = 0
				m.Viewport.GotoTop()
				m.Viewport.SetContent(m.historyListView())
			} else {
				m.HistoryGGPressed = true
			}
			return m, nil, true
		case "G":
			if len(m.FilteredHistoryItems) > 0 {
				m.HistoryCussorPos = len(m.FilteredHistoryItems) - 1
				m.Viewport.GotoBottom()
				m.Viewport.SetContent(m.historyListView())
			}
			return m, nil, true
		case "d":
			m.scrollHistoryHalfPage(true)
			return m, nil, true
		case "u":
			m.scrollHistoryHalfPage(false)
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
	m.centerHistoryViewport()
	m.Viewport.SetContent(m.historyListView())
}

func (m *Model) scrollHistoryHalfPage(down bool) {
	if len(m.FilteredHistoryItems) == 0 {
		return
	}
	scrollAmount := m.Viewport.Height / 2
	m.HistoryCussorPos = cursorPosAfterScroll(m.HistoryCussorPos, scrollAmount, len(m.FilteredHistoryItems), down)
	m.centerHistoryViewport()
	m.Viewport.SetContent(m.historyListView())
}

func (m *Model) centerHistoryViewport() {
	if len(m.FilteredHistoryItems) == 0 {
		return
	}

	halfHeight := m.Viewport.Height / 2
	targetOffset := m.HistoryCussorPos - halfHeight

	maxOffset := len(m.FilteredHistoryItems) - m.Viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if targetOffset < 0 {
		targetOffset = 0
	} else if targetOffset > maxOffset {
		targetOffset = maxOffset
	}

	m.Viewport.SetYOffset(targetOffset)
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
