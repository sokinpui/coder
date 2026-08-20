package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

type FinderModel struct {
	TextInput  textinput.Model
	AllItems   []string
	FoundItems []string
	Cursor     int
	Width      int
	Height     int
}

func NewFinder() FinderModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return FinderModel{
		TextInput: ti,
	}
}

func (m FinderModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *FinderModel) updateFoundItems() {
	query := m.TextInput.Value()
	if query == "" {
		m.FoundItems = m.AllItems
	} else {
		matches := fuzzy.Find(query, m.AllItems)
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].Index < matches[j].Index
		})
		m.FoundItems = make([]string, len(matches))
		for i, match := range matches {
			m.FoundItems[i] = match.Str
		}
	}
	if m.Cursor >= len(m.FoundItems) {
		m.Cursor = 0
	}
}

func (m FinderModel) View() string {
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
	end := min(start+maxItems, len(m.FoundItems))

	for i, item := range m.FoundItems[start:end] {
		actualIndex := i + start
		if actualIndex == m.Cursor {
			b.WriteString(paletteSelectedItemStyle.Render("▸ " + item))
		} else {
			b.WriteString(paletteItemStyle.Render("  " + item))
		}
		b.WriteString("\n")
	}

	return paletteContainerStyle.Width(m.Width).Render(b.String())
}

type FinderOverlay struct{}

func (f *FinderOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayFinder
}

func (f *FinderOverlay) View(main *Model) string {
	finderWidth := main.Width / 2
	finderHeight := main.Height / 2
	if finderWidth < 60 {
		finderWidth = 60
	}
	if finderHeight < 10 {
		finderHeight = 10
	}
	main.Finder.Width = finderWidth
	main.Finder.Height = finderHeight
	main.Finder.TextInput.Width = finderWidth - 4

	finderContent := main.Finder.View()
	if finderContent == "" {
		return main.View()
	}

	return OverlayCenter(finderContent, main.View())
}
