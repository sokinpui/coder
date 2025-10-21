package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) exitSearchMode() (Model, tea.Cmd, bool) {
	m.State = stateIdle
	m.SearchInput.Blur()
	m.TextArea.Focus()
	m.SearchQuery = ""
	m.currentMatchFirstOnLine = -1
	m.SearchResultLines = nil
	m.CurrentSearchResult = 0
	m.SearchableContent = nil
	m.Viewport.SetContent(m.renderConversation(false)) // Re-render to remove highlights
	return m, textarea.Blink, true
}

func (m Model) calculateCurrentMatchHighlightIndex() Model {
	m.currentMatchFirstOnLine = -1 // Reset

	if m.SearchQuery == "" || len(m.SearchResultLines) == 0 || len(m.SearchableContent) == 0 {
		return m
	}

	lines := m.SearchableContent

	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(m.SearchQuery))
	if err != nil {
		return m
	}

	currentLineNum := m.SearchResultLines[m.CurrentSearchResult]
	globalMatchCount := 0
	for i, line := range lines {
		matches := re.FindAllStringIndex(line, -1)
		if i < currentLineNum {
			globalMatchCount += len(matches)
		} else if i == currentLineNum {
			if len(matches) > 0 {
				m.currentMatchFirstOnLine = globalMatchCount
			}
			break
		}
	}
	return m
}

func (m Model) performSearch() Model {
	m.SearchResultLines = []int{}
	m.CurrentSearchResult = 0
	m.currentMatchFirstOnLine = -1 // Reset

	if m.SearchQuery == "" {
		return m
	}

	lines := m.SearchableContent

	query := strings.ToLower(m.SearchQuery)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			m.SearchResultLines = append(m.SearchResultLines, i)
		}
	}

	if len(m.SearchResultLines) > 0 {
		currentTopLine := m.Viewport.YOffset
		found := false
		for i, lineNum := range m.SearchResultLines {
			if lineNum >= currentTopLine {
				m.CurrentSearchResult = i
				found = true
				break
			}
		}
		if !found {
			m.CurrentSearchResult = 0
		}
		m = m.calculateCurrentMatchHighlightIndex()
		m.Viewport.SetYOffset(m.SearchResultLines[m.CurrentSearchResult])
	}

	return m
}

func (m Model) handleKeyPressSearching(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.isSearchDebouncing = false
		return m.exitSearchMode()

	case tea.KeyEnter:
		m.isSearchDebouncing = false
		if m.SearchQuery != "" && len(m.SearchResultLines) > 0 {
			m.State = stateSearchNav
			m.SearchInput.Blur()
			m.TextArea.Reset()
			return m, nil, true
		}
		return m.exitSearchMode()
	}

	originalQuery := m.SearchInput.Value()
	m.SearchInput, cmd = m.SearchInput.Update(msg)
	cmds = append(cmds, cmd)
	if m.SearchInput.Value() != originalQuery && m.State == stateSearching {
		m.SearchQuery = m.SearchInput.Value()
		m.isSearchDebouncing = true
		cmds = append(cmds, searchDebounceTick())
	}

	return m, tea.Batch(cmds...), true
}

func (m Model) handleKeyPressSearchNav(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyRunes:
		switch msg.String() {
		case "n":
			if len(m.SearchResultLines) > 0 {
				m.CurrentSearchResult = (m.CurrentSearchResult + 1) % len(m.SearchResultLines)
				m = m.calculateCurrentMatchHighlightIndex()
				m.Viewport.SetYOffset(m.SearchResultLines[m.CurrentSearchResult])
				m.Viewport.SetContent(m.renderConversation(false))
			}
			return m, nil, true
		case "N":
			if len(m.SearchResultLines) > 0 {
				m.CurrentSearchResult--
				if m.CurrentSearchResult < 0 {
					m.CurrentSearchResult = len(m.SearchResultLines) - 1
				}
				m = m.calculateCurrentMatchHighlightIndex()
				m.Viewport.SetYOffset(m.SearchResultLines[m.CurrentSearchResult])
				m.Viewport.SetContent(m.renderConversation(false))
			}
			return m, nil, true
		case "/":
			// Start a new search
			m.State = stateSearching
			m.SearchInput.Focus()
			m.SearchInput.Reset()
			m.TextArea.Blur()
			return m, textinput.Blink, true
		}

	case tea.KeyEsc, tea.KeyCtrlC:
		return m.exitSearchMode()
	}

	// Any other key press exits search nav mode and gets handled by the idle state.
	// We need to pass the key press on.
	m.State = stateIdle
	m.currentMatchFirstOnLine = -1
	m.SearchQuery = ""
	m.SearchResultLines = nil
	m.CurrentSearchResult = 0
	m.Viewport.SetContent(m.renderConversation(false)) // remove highlights
	m.TextArea.Focus()

	// Let the idle handler process the key.
	return m, textarea.Blink, false
}
