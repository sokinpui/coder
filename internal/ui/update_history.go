package ui

import (
	"coder/internal/core"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPressHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.historyItems = nil
		if m.isStreaming {
			// Return to the generation view
			messages := m.session.GetMessages()
			if len(messages) > 0 && messages[len(messages)-1].Type == core.AIMessage && messages[len(messages)-1].Content == "" {
				m.state = stateThinking
			} else {
				m.state = stateGenerating
			}
			m.viewport.SetContent(m.renderConversation())
			// Re-issue commands needed for generation state
			return m, tea.Batch(listenForStream(m.streamSub), renderTick(), m.spinner.Tick), true
		} else {
			// Return to idle
			m.state = stateIdle
			m.textArea.Focus()
			m.viewport.SetContent(m.renderConversation())
			return m, textarea.Blink, true
		}

	case tea.KeyEnter:
		if len(m.historyItems) == 0 || m.historySelectCursor >= len(m.historyItems) {
			return m, nil, true
		}
		selectedItem := m.historyItems[m.historySelectCursor]
		if m.isStreaming {
			m.session.CancelGeneration()
			m.isStreaming = false // Prevent streamFinishedMsg from running
			m.streamSub = nil
		}
		return m, loadConversationCmd(m.session, selectedItem.Filename), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if m.historySelectCursor < len(m.historyItems)-1 {
				m.historySelectCursor++
				headerHeight := 10 // offset to bottom
				cursorLine := m.historySelectCursor + headerHeight
				if cursorLine >= m.viewport.YOffset+m.viewport.Height {
					m.viewport.LineDown(1)
				}
				m.viewport.SetContent(m.historyView()) // Re-render to update selection highlight
			}
			return m, nil, true
		case "k":
			if m.historySelectCursor > 0 {
				m.historySelectCursor--
				headerHeight := -10 // offset to top
				cursorLine := m.historySelectCursor + headerHeight
				if cursorLine < m.viewport.YOffset {
					m.viewport.LineUp(1)
				}
				m.viewport.SetContent(m.historyView()) // Re-render to update selection highlight
			}
			return m, nil, true
		}
	}
	return m, nil, true
}
