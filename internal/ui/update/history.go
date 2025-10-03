package update

import (
	"coder/internal/core"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPressHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.HistoryItems = nil
		if m.IsStreaming {
			// Return to the generation view
			messages := m.Session.GetMessages()
			if len(messages) > 0 && messages[len(messages)-1].Type == core.AIMessage && messages[len(messages)-1].Content == "" {
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
		if len(m.HistoryItems) == 0 || m.HistorySelectCursor >= len(m.HistoryItems) {
			return m, nil, true
		}
		selectedItem := m.HistoryItems[m.HistorySelectCursor]
		if m.IsStreaming {
			m.Session.CancelGeneration()
			m.IsStreaming = false // Prevent streamFinishedMsg from running
			m.StreamSub = nil
		}
		return m, loadConversationCmd(m.Session, selectedItem.Filename), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if m.HistorySelectCursor < len(m.HistoryItems)-1 {
				m.HistorySelectCursor++
				headerHeight := 10 // offset to bottom
				cursorLine := m.HistorySelectCursor + headerHeight
				if cursorLine >= m.Viewport.YOffset+m.Viewport.Height {
					m.Viewport.LineDown(1)
				}
				m.Viewport.SetContent(m.historyView()) // Re-render to update selection highlight
			}
			return m, nil, true
		case "k":
			if m.HistorySelectCursor > 0 {
				m.HistorySelectCursor--
				headerHeight := -10 // offset to top
				cursorLine := m.HistorySelectCursor + headerHeight
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
