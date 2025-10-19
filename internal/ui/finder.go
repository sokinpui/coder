package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rmhubbert/bubbletea-overlay"
	"github.com/sahilm/fuzzy"
)

// FinderModel is the model for the fuzzy finder.
type FinderModel struct {
	TextInput  textinput.Model
	AllItems   []string
	FoundItems []string
	Cursor     int
	Width      int
	Height     int
	Visible    bool
}

// NewFinder creates a new finder model.
func NewFinder() FinderModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return FinderModel{
		TextInput: ti,
		Visible:   false,
	}
}

func (m FinderModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FinderModel) Update(msg tea.Msg) (FinderModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.Visible = false
			m.TextInput.Blur()
			m.TextInput.Reset()
			return m, nil

		case tea.KeyUp, tea.KeyCtrlP:
			if m.Cursor > 0 {
				m.Cursor--
			}
			return m, nil

		case tea.KeyDown, tea.KeyCtrlN:
			if m.Cursor < len(m.FoundItems)-1 {
				m.Cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if len(m.FoundItems) > 0 && m.Cursor < len(m.FoundItems) {
				selected := m.FoundItems[m.Cursor]
				m.Visible = false
				m.TextInput.Blur()
				m.TextInput.Reset()
				return m, func() tea.Msg { return finderResultMsg{result: selected} }
			}
			return m, nil
		}
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	m.updateFoundItems()

	return m, cmd
}

func (m *FinderModel) updateFoundItems() {
	query := m.TextInput.Value()
	if query == "" {
		m.FoundItems = m.AllItems
	} else {
		matches := fuzzy.Find(query, m.AllItems)
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

// FinderOverlay implements the Overlay interface for the fuzzy finder.
type FinderOverlay struct{}

func (f *FinderOverlay) IsVisible(main *Model) bool {
	return main.State == stateFinder
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

	finderModel := simpleModel{content: finderContent}

	overlayModel := overlay.New(
		finderModel,
		main,
		overlay.Center,
		overlay.Center,
		0,
		0,
	)
	return overlayModel.View()
}
