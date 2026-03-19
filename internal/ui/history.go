package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/types"
)

type HistoryTab int

const (
	TabHistory HistoryTab = iota
	TabActive
)

type HistoryModel struct {
	Items         []history.ConversationInfo
	FilteredItems []history.ConversationInfo
	ActiveItems   []history.ConversationInfo
	CursorPos     int
	SearchInput   textinput.Model
	IsSearching   bool
	GGPressed     bool
	Tab           HistoryTab
}

func NewHistory() HistoryModel {
	hsi := textinput.New()
	hsi.Placeholder = "Fuzzy search..."
	hsi.Prompt = "/"
	hsi.Width = 50
	hsi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	return HistoryModel{
		SearchInput: hsi,
		Tab:         TabHistory,
	}
}

func (m Model) handleKeyPressHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.History.IsSearching {
		switch msg.Type {
		case tea.KeyEnter:
			m.History.IsSearching = false
			m.History.SearchInput.Blur()
			m.Chat.Viewport.SetContent(m.historyListView())
			return m, nil, true

		case tea.KeyEsc, tea.KeyCtrlC:
			m.History.IsSearching = false
			m.History.SearchInput.Blur()
			m.History.SearchInput.Reset()
			m.updateHistoryFilter()
			m.updateActiveFilter()
			m.Chat.Viewport.SetContent(m.historyListView())
			return m, nil, true
		}

		var cmd tea.Cmd
		m.History.SearchInput, cmd = m.History.SearchInput.Update(msg)
		m.updateHistoryFilter()
		m.updateActiveFilter()
		m.Chat.Viewport.SetContent(m.historyListView())
		return m, cmd, true
	}

	prevGGPressed := m.History.GGPressed
	m.History.GGPressed = false // Reset by default

	switch msg.Type {
	case tea.KeyTab, tea.KeyShiftTab:
		if m.History.Tab == TabHistory {
			m.History.Tab = TabActive
		} else {
			m.History.Tab = TabHistory
		}
		m.History.CursorPos = 0
		m.updateHistoryFilter()
		m.updateActiveFilter()
		m.centerHistoryViewport()
		m.Chat.Viewport.SetContent(m.historyListView())
		return m, nil, true

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
		m.History.Items = nil
		if m.Chat.IsStreaming {
			// Return to the generation view
			messages := m.Session.GetMessages()
			if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
				m.State = stateThinking
			} else {
				m.State = stateGenerating
			}
			delay := m.Session.GetConfig().Generation.StreamDelay
			m.Chat.Viewport.SetContent(m.renderConversation())
			// Re-issue commands needed for generation state
			return m, tea.Batch(listenForStream(m.Chat.StreamSub), streamAnimeCmd(delay), m.Chat.Spinner.Tick), true
		} else {
			// Return to idle
			m.State = stateIdle
			m.Chat.TextArea.Focus()
			m.Chat.Viewport.SetContent(m.renderConversation())
			return m, textarea.Blink, true
		}

	case tea.KeyEnter:
		currentItems := m.getHistoryCurrentList()
		if len(currentItems) == 0 || m.History.CursorPos >= len(currentItems) {
			return m, nil, true
		}
		selectedItem := currentItems[m.History.CursorPos]

		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false // Prevent streamFinishedMsg from running
			m.Chat.StreamSub = nil
		}

		if m.History.Tab == TabActive {
			// Load the actual pointer from the active session list
			targetSess := m.ActiveSessions[m.History.CursorPos]
			return m, func() tea.Msg { return switchActiveSessionMsg{sess: targetSess} }, true
		}

		return m, loadConversationCmd(m.Session, selectedItem.Filename), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "q":
			return m.handleKeyPressHistory(tea.KeyMsg{Type: tea.KeyEsc})
		case "/":
			m.History.IsSearching = true
			m.History.SearchInput.Focus()
			m.History.SearchInput.Reset()
			m.updateHistoryFilter()
			m.updateActiveFilter()
			m.Chat.Viewport.SetContent(m.historyListView())
			return m, nil, true
		case "g":
			if prevGGPressed {
				m.History.CursorPos = 0
				m.Chat.Viewport.GotoTop()
				m.Chat.Viewport.SetContent(m.historyListView())
			} else {
				m.History.GGPressed = true
			}
			return m, nil, true
		case "G":
			currentItems := m.getHistoryCurrentList()
			if len(currentItems) > 0 {
				m.History.CursorPos = len(currentItems) - 1
				m.Chat.Viewport.GotoBottom()
				m.Chat.Viewport.SetContent(m.historyListView())
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

func (m Model) getHistoryCurrentList() []history.ConversationInfo {
	if m.History.Tab == TabHistory {
		return m.History.FilteredItems
	}
	return m.History.ActiveItems
}

func (m *Model) moveHistoryCursor(delta int) {
	currentItems := m.getHistoryCurrentList()
	newPos := m.History.CursorPos + delta
	if newPos < 0 || newPos >= len(currentItems) {
		return
	}

	m.History.CursorPos = newPos
	m.centerHistoryViewport()
	m.Chat.Viewport.SetContent(m.historyListView())
}

func (m *Model) scrollHistoryHalfPage(down bool) {
	currentItems := m.getHistoryCurrentList()
	if len(currentItems) == 0 {
		return
	}
	scrollAmount := m.Chat.Viewport.Height / 2
	m.History.CursorPos = cursorPosAfterScroll(m.History.CursorPos, scrollAmount, len(currentItems), down)
	m.centerHistoryViewport()
	m.Chat.Viewport.SetContent(m.historyListView())
}

func (m *Model) centerHistoryViewport() {
	currentItems := m.getHistoryCurrentList()
	if len(currentItems) == 0 {
		return
	}

	halfHeight := m.Chat.Viewport.Height / 2
	targetOffset := m.History.CursorPos - halfHeight

	maxOffset := len(currentItems) - m.Chat.Viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if targetOffset < 0 {
		targetOffset = 0
	} else if targetOffset > maxOffset {
		targetOffset = maxOffset
	}

	m.Chat.Viewport.SetYOffset(targetOffset)
}

func (m *Model) updateHistoryFilter() {
	query := m.History.SearchInput.Value()
	if query == "" {
		m.History.FilteredItems = m.History.Items
		return
	}

	targets := make([]string, len(m.History.Items))
	for i, item := range m.History.Items {
		targets[i] = item.Title + " " + item.Filename
	}

	matches := fuzzy.Find(query, targets)
	var filtered []history.ConversationInfo
	for _, match := range matches {
		filtered = append(filtered, m.History.Items[match.Index])
	}

	m.History.FilteredItems = filtered
	if m.History.CursorPos >= len(m.History.FilteredItems) {
		m.History.CursorPos = max(0, len(m.History.FilteredItems)-1)
	}
}

func (m *Model) updateActiveFilter() {
	var activeItems []history.ConversationInfo
	for _, sess := range m.ActiveSessions {
		activeItems = append(activeItems, history.ConversationInfo{
			Title:      sess.GetTitle(),
			Filename:   sess.GetHistoryFilename(),
			ModifiedAt: time.Now(), // Active sessions are "now"
		})
	}

	query := m.History.SearchInput.Value()
	if query == "" {
		m.History.ActiveItems = activeItems
		return
	}

	targets := make([]string, len(activeItems))
	for i, item := range activeItems {
		targets[i] = item.Title
	}

	matches := fuzzy.Find(query, targets)
	var filtered []history.ConversationInfo
	for _, match := range matches {
		filtered = append(filtered, activeItems[match.Index])
	}

	m.History.ActiveItems = filtered
	if m.History.CursorPos >= len(m.History.ActiveItems) {
		m.History.CursorPos = max(0, len(m.History.ActiveItems)-1)
	}
}

func (m Model) historyHeaderView() string {
	var b strings.Builder
	if m.History.IsSearching {
		b.WriteString("Search History: ")
		b.WriteString(m.History.SearchInput.View())
		b.WriteString("\n\n")
	} else {
		b.WriteString("Select a conversation to load (type / to search):\n")
	}

	historyTabStr := "[ History ]"
	activeTabStr := "[ Active ]"

	if m.History.Tab == TabHistory {
		historyTabStr = activeTabStyle.Render(historyTabStr)
		activeTabStr = tabStyle.Render(activeTabStr)
	} else {
		historyTabStr = tabStyle.Render(historyTabStr)
		activeTabStr = activeTabStyle.Render(activeTabStr)
	}

	b.WriteString(fmt.Sprintf("%s  %s\n\n", historyTabStr, activeTabStr))

	return b.String()
}

func (m Model) historyListView() string {
	var b strings.Builder
	currentItems := m.getHistoryCurrentList()

	if len(currentItems) == 0 {
		b.WriteString("  No matching history found.")
		return b.String()
	}

	for i, item := range currentItems {
		title := item.Title
		dateStr := ""
		if m.History.Tab == TabHistory {
			dateStr = fmt.Sprintf(" (%s)", item.ModifiedAt.Format("2006-01-02 15:04"))
		}
		if i == m.History.CursorPos {
			b.WriteString(paletteSelectedItemStyle.Render(fmt.Sprintf("▸  %s%s", title, dateStr)))
		} else {
			b.WriteString(paletteItemStyle.Render(fmt.Sprintf("   %s%s", title, dateStr)))
		}
		b.WriteString("\n")
	}

	return b.String()
}
