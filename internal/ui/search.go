package ui

import (
	"coder/internal/types"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rmhubbert/bubbletea-overlay"
	"github.com/sahilm/fuzzy"
)

type SearchItem struct {
	MsgIndex int
	LineNum  int
	Text     string
}

// SearchModel is the model for the fuzzy finder.
type SearchModel struct {
	TextInput  textinput.Model
	AllItems   []SearchItem
	FoundItems []fuzzy.Match
	Cursor     int
	Width      int
	Height     int
	Visible    bool
}

// NewSearch creates a new search model.
func NewSearch() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search conversation..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return SearchModel{
		TextInput: ti,
		Visible:   false,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.Visible = false
			m.TextInput.Blur()
			m.TextInput.Reset()
			return m, nil

		case tea.KeyUp, tea.KeyCtrlP, tea.KeyCtrlK:
			if m.Cursor > 0 {
				m.Cursor--
			}
			return m, nil

		case tea.KeyDown, tea.KeyCtrlN, tea.KeyCtrlJ:
			if m.Cursor < len(m.FoundItems)-1 {
				m.Cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if len(m.FoundItems) > 0 && m.Cursor < len(m.FoundItems) {
				selected := m.AllItems[m.FoundItems[m.Cursor].Index]
				m.Visible = false
				m.TextInput.Blur()
				// m.TextInput.Reset()
				return m, func() tea.Msg { return searchResultMsg{item: selected} }
			}
			return m, nil
		}
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	m.updateFoundItems()

	return m, cmd
}

func (m *SearchModel) getSourceItems() []string {
	var source []string
	for _, item := range m.AllItems {
		source = append(source, item.Text)
	}
	return source
}

func (m *SearchModel) updateFoundItems() {
	query := m.TextInput.Value()
	if query == "" {
		m.FoundItems = nil
	} else {
		m.FoundItems = fuzzy.Find(query, m.getSourceItems())
	}
	if m.Cursor >= len(m.FoundItems) {
		m.Cursor = 0
	}
}

func (m SearchModel) View() string {
	if !m.Visible {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.TextInput.View())
	b.WriteString("\n\n")

	maxItems := m.Height - 4 // account for input and padding
	if maxItems < 1 {
		maxItems = 5
	}

	start := 0
	if m.Cursor >= maxItems {
		start = m.Cursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(m.FoundItems) {
		end = len(m.FoundItems)
	}

	for i, match := range m.FoundItems[start:end] {
		actualIndex := i + start
		item := m.AllItems[match.Index]
		line := fmt.Sprintf("Msg %d: %s", item.MsgIndex, item.Text)
		if actualIndex == m.Cursor {
			b.WriteString(paletteSelectedItemStyle.Render("▸ " + line))
		} else {
			b.WriteString(paletteItemStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	return paletteContainerStyle.Width(m.Width).Render(b.String())
}

// SearchOverlay implements the Overlay interface for the fuzzy finder.
type SearchOverlay struct{}

func (f *SearchOverlay) IsVisible(main *Model) bool {
	return main.State == stateSearch
}

func (f *SearchOverlay) View(main *Model) string {
	searchWidth := main.Width / 2
	searchHeight := main.Height / 2
	if searchWidth < 60 {
		searchWidth = 60
	}
	if searchHeight < 10 {
		searchHeight = 10
	}
	main.Search.Width = searchWidth
	main.Search.Height = searchHeight
	main.Search.TextInput.Width = searchWidth - 4

	searchContent := main.Search.View()
	if searchContent == "" {
		return main.View()
	}

	searchModel := simpleModel{content: searchContent}

	overlayModel := overlay.New(
		searchModel,
		main,
		overlay.Center,
		overlay.Center,
		0,
		0,
	)
	return overlayModel.View()
}

func (m *Model) collectSearchableMessages() []SearchItem {
	var items []SearchItem
	messages := m.Session.GetMessages()
	for i, msg := range messages {
		if msg.Type == types.UserMessage || msg.Type == types.AIMessage {
			lines := strings.Split(msg.Content, "\n")
			for lineNum, line := range lines {
				if strings.TrimSpace(line) != "" {
					items = append(items, SearchItem{
						MsgIndex: i,
						LineNum:  lineNum,
						Text:     line,
					})
				}
			}
		}
	}
	return items
}
