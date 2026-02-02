package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/types"
	tea "github.com/charmbracelet/bubbletea"
)

type HistoryModel struct {
	Items         []history.ConversationInfo
	FilteredItems []history.ConversationInfo
	CursorPos     int
	SearchInput   textinput.Model
	IsSearching   bool
	GGPressed     bool
}

func NewHistory() HistoryModel {
	hsi := textinput.New()
	hsi.Placeholder = "Fuzzy search..."
	hsi.Prompt = "/"
	hsi.Width = 50
	hsi.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	return HistoryModel{
		SearchInput: hsi,
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
			m.Chat.Viewport.SetContent(m.historyListView())
			return m, nil, true
		}

		var cmd tea.Cmd
		m.History.SearchInput, cmd = m.History.SearchInput.Update(msg)
		m.updateHistoryFilter()
		m.Chat.Viewport.SetContent(m.historyListView())
		return m, cmd, true
	}

	prevGGPressed := m.History.GGPressed
	m.History.GGPressed = false // Reset by default

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
		if len(m.History.FilteredItems) == 0 || m.History.CursorPos >= len(m.History.FilteredItems) {
			return m, nil, true
		}
		selectedItem := m.History.FilteredItems[m.History.CursorPos]
		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false // Prevent streamFinishedMsg from running
			m.Chat.StreamSub = nil
		}
		return m, loadConversationCmd(m.Session, selectedItem.Filename), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "/":
			m.History.IsSearching = true
			m.History.SearchInput.Focus()
			m.History.SearchInput.Reset()
			m.updateHistoryFilter()
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
			if len(m.History.FilteredItems) > 0 {
				m.History.CursorPos = len(m.History.FilteredItems) - 1
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

func (m *Model) moveHistoryCursor(delta int) {
	newPos := m.History.CursorPos + delta
	if newPos < 0 || newPos >= len(m.History.FilteredItems) {
		return
	}

	m.History.CursorPos = newPos
	m.centerHistoryViewport()
	m.Chat.Viewport.SetContent(m.historyListView())
}

func (m *Model) scrollHistoryHalfPage(down bool) {
	if len(m.History.FilteredItems) == 0 {
		return
	}
	scrollAmount := m.Chat.Viewport.Height / 2
	m.History.CursorPos = cursorPosAfterScroll(m.History.CursorPos, scrollAmount, len(m.History.FilteredItems), down)
	m.centerHistoryViewport()
	m.Chat.Viewport.SetContent(m.historyListView())
}

func (m *Model) centerHistoryViewport() {
	if len(m.History.FilteredItems) == 0 {
		return
	}

	halfHeight := m.Chat.Viewport.Height / 2
	targetOffset := m.History.CursorPos - halfHeight

	maxOffset := len(m.History.FilteredItems) - m.Chat.Viewport.Height
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

func (m Model) historyHeaderView() string {
	var b strings.Builder
	if m.History.IsSearching {
		b.WriteString("Search History: ")
		b.WriteString(m.History.SearchInput.View())
		b.WriteString("\n\n")
	} else {
		b.WriteString("Select a conversation to load (type / to search):\n\n")
	}
	return b.String()
}

func (m Model) historyListView() string {
	var b strings.Builder

	if len(m.History.FilteredItems) == 0 {
		b.WriteString("  No matching history found.")
		return b.String()
	}

	for i, item := range m.History.FilteredItems {
		title := item.Title
		date := fmt.Sprintf("(%s)", item.ModifiedAt.Format("2006-01-02 15:04"))
		if i == m.History.CursorPos {
			b.WriteString(paletteSelectedItemStyle.Render("▸  " + title))
			b.WriteString(paletteItemStyle.Render(" " + date))
		} else {
			b.WriteString(paletteItemStyle.Render("   " + title + " " + date))
		}
		b.WriteString("\n")
	}

	return b.String()
}
