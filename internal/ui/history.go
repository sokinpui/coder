package ui

import (
	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPressHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	prevGGPressed := m.HistoryGGPressed
	m.HistoryGGPressed = false // Reset by default

	switch msg.Type {
	case tea.KeyCtrlD:
		if len(m.HistoryItems) == 0 {
			return m, nil, true
		}
		scrollAmount := m.Viewport.Height / 2
		m.Viewport.HalfPageDown()
		m.HistoryCussorPos = cursorPosAfterScroll(m.HistoryCussorPos, scrollAmount, len(m.HistoryItems), true)
		m.Viewport.SetContent(m.historyView())
		return m, nil, true
	case tea.KeyCtrlU:
		if len(m.HistoryItems) == 0 {
			return m, nil, true
		}
		scrollAmount := m.Viewport.Height / 2
		m.Viewport.HalfPageUp()
		m.HistoryCussorPos = cursorPosAfterScroll(m.HistoryCussorPos, scrollAmount, len(m.HistoryItems), false)
		m.Viewport.SetContent(m.historyView())
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
			m.Viewport.SetContent(m.renderConversation())
			// Re-issue commands needed for generation state
			return m, tea.Batch(listenForStream(m.StreamSub), renderTick(), m.Spinner.Tick), true
		} else {
			// Return to idle
			m.State = stateIdle
			m.TextArea.Focus()
			m.Viewport.SetContent(m.renderConversation())
			return m, textarea.Blink, true
		}

	case tea.KeyEnter:
		if len(m.HistoryItems) == 0 || m.HistoryCussorPos >= len(m.HistoryItems) {
			return m, nil, true
		}
		selectedItem := m.HistoryItems[m.HistoryCussorPos]
		if m.IsStreaming {
			m.Session.CancelGeneration()
			m.IsStreaming = false // Prevent streamFinishedMsg from running
			m.StreamSub = nil
		}
		return m, loadConversationCmd(m.Session, selectedItem.Filename), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
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
			if len(m.HistoryItems) > 0 {
				m.HistoryCussorPos = len(m.HistoryItems) - 1
				m.Viewport.GotoBottom()
				m.Viewport.SetContent(m.historyView())
			}
			return m, nil, true
		case "j":
			if m.HistoryCussorPos < len(m.HistoryItems)-1 {
				m.HistoryCussorPos++
				headerHeight := 10 // offset to bottom
				cursorLine := m.HistoryCussorPos + headerHeight
				if cursorLine >= m.Viewport.YOffset+m.Viewport.Height {
					m.Viewport.LineDown(1)
				}
				m.Viewport.SetContent(m.historyView()) // Re-render to update selection highlight
			}
			return m, nil, true
		case "k":
			if m.HistoryCussorPos > 0 {
				m.HistoryCussorPos--
				headerHeight := -10 // offset to top
				cursorLine := m.HistoryCussorPos + headerHeight
				if cursorLine < m.Viewport.YOffset {
					m.Viewport.LineUp(1)
				}
				m.Viewport.SetContent(m.historyView()) // Re-render to update selection highlight
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
