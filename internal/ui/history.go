package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
	"github.com/rmhubbert/bubbletea-overlay"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/history"
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
	HistoryCursor int
	ActiveCursor  int
	SearchInput   textinput.Model
	IsSearching   bool
	GGPressed     bool
	Tab           HistoryTab
	Width         int
	Height        int
}

func NewHistory() HistoryModel {
	hsi := textinput.New()
	hsi.Placeholder = "Fuzzy search..."
	hsi.Prompt = ""
	hsi.Width = 50
	hsi.PlaceholderStyle = searchPlaceholderStyle

	return HistoryModel{
		SearchInput: hsi,
		Tab:         TabHistory,
	}
}

type HistoryOverlay struct{}

func (h *HistoryOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayHistory
}

func (h *HistoryOverlay) View(main *Model) string {
	historyWidth := min(90, max(50, main.Width-4))
	historyHeight := min(30, max(12, main.Height-4))
	main.History.Width = historyWidth
	main.History.Height = historyHeight
	main.History.SearchInput.Width = historyWidth - 20

	content := main.History.View(main)
	if content == "" {
		return main.View()
	}

	return overlay.New(
		simpleModel{content: content},
		main,
		overlay.Center,
		overlay.Center,
		0, 0,
	).View()
}

func (m HistoryModel) View(main *Model) string {
	historyTabStr := "[ History ]"
	activeTabStr := "[ Active ]"

	if m.Tab == TabHistory {
		historyTabStr = activeTabStyle.Render(historyTabStr)
		activeTabStr = tabStyle.Render(activeTabStr)
	} else {
		historyTabStr = tabStyle.Render(historyTabStr)
		activeTabStr = activeTabStyle.Render(activeTabStr)
	}

	var header strings.Builder
	header.WriteString(fmt.Sprintf("%s  %s", historyTabStr, activeTabStr))
	if m.IsSearching {
		header.WriteString("   Search: " + m.SearchInput.View())
	}
	header.WriteString("\n")

	var listBuf strings.Builder
	currentItems := main.getHistoryCurrentList()

	if len(currentItems) == 0 {
		listBuf.WriteString("  No matching history found.\n")
	} else {
		maxItems := max(5, m.Height-6)
		start := 0
		if m.CursorPos >= maxItems {
			start = m.CursorPos - maxItems + 1
		}
		end := min(start+maxItems, len(currentItems))

		for i := start; i < end; i++ {
			item := currentItems[i]
			title := item.Title
			dateStr := ""
			if m.Tab == TabHistory {
				dateStr = fmt.Sprintf(" (%s)", item.CreatedAt.Format("2006-01-02 15:04"))
			}

			isCurrent := false
			if m.Tab == TabActive && item.ID == main.Session.ID {
				isCurrent = true
			} else if m.Tab == TabHistory && item.Filename != "" && item.Filename == main.Session.GetHistoryFilename() {
				isCurrent = true
			}

			cursor := "  "
			if i == m.CursorPos {
				cursor = "▸ "
			}

			marker := " "
			if isCurrent {
				marker = "*"
			}

			prefix := fmt.Sprintf("%s%s", cursor, marker)
			line := fmt.Sprintf("%s %s%s", prefix, title, dateStr)
			if i == m.CursorPos {
				listBuf.WriteString(paletteSelectedItemStyle.Render(line))
			} else {
				listBuf.WriteString(paletteItemStyle.Render(line))
			}
			listBuf.WriteString("\n")
		}
	}

	footer := paletteHeaderStyle.Render("── [Esc/q: close | /: search | Tab: switch tab | Enter: load] ──")
	content := lipgloss.JoinVertical(lipgloss.Left, header.String(), listBuf.String(), footer)
	return paletteContainerStyle.Width(m.Width).Render(content)
}

func (m Model) handleKeyPressHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	keyStr := msg.String()
	km := m.Session.GetConfig().Keymap

	if m.History.IsSearching {
		switch msg.Type {
		case tea.KeyUp, tea.KeyCtrlK, tea.KeyCtrlP:
			m.moveHistoryCursor(-1)
			return m, nil, true

		case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
			m.moveHistoryCursor(1)
			return m, nil, true

		case tea.KeyEnter:
			m.History.IsSearching = false
			m.History.SearchInput.Blur()
			return m, nil, true

		case tea.KeyEsc, tea.KeyCtrlC:
			m.History.IsSearching = false
			m.History.SearchInput.Blur()
			m.History.SearchInput.Reset()
			m.updateHistoryFilter()
			m.updateActiveFilter()
			return m, nil, true
		}

		var cmd tea.Cmd
		m.History.SearchInput, cmd = m.History.SearchInput.Update(msg)
		m.updateHistoryFilter()
		m.updateActiveFilter()
		return m, cmd, true
	}

	prevGGPressed := m.History.GGPressed
	m.History.GGPressed = false

	switch keyStr {
	case km.ScrollDown:
		m.scrollHistoryHalfPage(true)
		return m, nil, true
	case km.ScrollUp:
		m.scrollHistoryHalfPage(false)
		return m, nil, true
	}

	switch msg.Type {
	case tea.KeyTab, tea.KeyShiftTab:
		target := TabActive
		if m.History.Tab == TabActive {
			target = TabHistory
		}
		m.switchTab(target)
		return m, nil, true

	case tea.KeyUp, tea.KeyCtrlK:
		m.moveHistoryCursor(-1)
		return m, nil, true

	case tea.KeyDown, tea.KeyCtrlJ:
		m.moveHistoryCursor(1)
		return m, nil, true

	case tea.KeyEsc, tea.KeyCtrlC:
		m.ActiveOverlay = overlayNone
		m.History.Items = nil
		m.History.IsSearching = false
		m.History.SearchInput.Blur()
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true

	case tea.KeyEnter:
		currentItems := m.getHistoryCurrentList()
		if len(currentItems) == 0 || m.History.CursorPos >= len(currentItems) {
			return m, nil, true
		}
		selectedItem := currentItems[m.History.CursorPos]

		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false
			m.Chat.StreamSub = nil
		}

		m.ActiveOverlay = overlayNone
		m.History.IsSearching = false
		m.History.SearchInput.Blur()

		if m.History.Tab == TabActive {
			return m, m.switchSessionByID(selectedItem.ID), true
		}

		for _, sess := range m.ActiveSessions {
			if sess.GetHistoryFilename() == selectedItem.Filename {
				return m, m.switchSessionByID(sess.ID), true
			}
		}

		return m, loadConversationCmd(m.Session, selectedItem.Filename), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case km.HistoryView.Exit:
			return m.handleKeyPressHistory(tea.KeyMsg{Type: tea.KeyEsc})
		case km.HistoryView.Search:
			m.History.IsSearching = true
			m.History.SearchInput.Focus()
			m.History.SearchInput.Reset()
			m.updateHistoryFilter()
			m.updateActiveFilter()
			return m, nil, true
		case km.HistoryView.HistoryTab:
			target := TabHistory
			if m.History.Tab == TabHistory {
				target = TabActive
			}
			m.switchTab(target)
			return m, nil, true
		case km.HistoryView.ActiveTab:
			target := TabActive
			if m.History.Tab == TabActive {
				target = TabHistory
			}
			m.switchTab(target)
			return m, nil, true
		case km.HistoryView.Top:
			if prevGGPressed || km.HistoryView.Top != "g" {
				m.History.CursorPos = 0
			} else {
				m.History.GGPressed = true
			}
			return m, nil, true
		case km.HistoryView.Bottom:
			currentItems := m.getHistoryCurrentList()
			if len(currentItems) > 0 {
				m.History.CursorPos = len(currentItems) - 1
			}
			return m, nil, true
		case km.HistoryView.HalfPageDown:
			m.scrollHistoryHalfPage(true)
			return m, nil, true
		case km.HistoryView.HalfPageUp:
			m.scrollHistoryHalfPage(false)
			return m, nil, true
		case km.HistoryView.Down:
			m.moveHistoryCursor(1)
			return m, nil, true
		case km.HistoryView.Up:
			m.moveHistoryCursor(-1)
			return m, nil, true
		}
	}
	return m, nil, true
}

func (m *Model) switchTab(target HistoryTab) {
	if m.History.Tab == target {
		return
	}

	if m.History.Tab == TabHistory {
		m.History.HistoryCursor = m.History.CursorPos
	} else {
		m.History.ActiveCursor = m.History.CursorPos
	}

	m.History.Tab = target
	m.History.CursorPos = m.History.HistoryCursor
	if target == TabActive {
		m.History.CursorPos = m.History.ActiveCursor
	}

	m.updateHistoryFilter()
	m.updateActiveFilter()
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
}

func (m *Model) scrollHistoryHalfPage(down bool) {
	currentItems := m.getHistoryCurrentList()
	if len(currentItems) == 0 {
		return
	}
	scrollAmount := max(1, m.History.Height/2)
	m.History.CursorPos = cursorPosAfterScroll(m.History.CursorPos, scrollAmount, len(currentItems), down)
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

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	m.History.FilteredItems = filtered
	if m.History.Tab == TabHistory {
		if m.History.CursorPos >= len(m.History.FilteredItems) {
			m.History.CursorPos = max(0, len(m.History.FilteredItems)-1)
		}
		m.History.HistoryCursor = m.History.CursorPos
	}
}

func (m *Model) updateActiveFilter() {
	var activeItems []history.ConversationInfo
	for i := len(m.ActiveSessions) - 1; i >= 0; i-- {
		sess := m.ActiveSessions[i]
		activeItems = append(activeItems, history.ConversationInfo{
			ID:         sess.ID,
			Title:      sess.GetTitle(),
			Filename:   sess.GetHistoryFilename(),
			CreatedAt:  sess.GetCreatedAt(),
			ModifiedAt: time.Now(),
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

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	m.History.ActiveItems = filtered
	if m.History.Tab == TabActive {
		if m.History.CursorPos >= len(m.History.ActiveItems) {
			m.History.CursorPos = max(0, len(m.History.ActiveItems)-1)
		}
		m.History.ActiveCursor = m.History.CursorPos
	}
}
