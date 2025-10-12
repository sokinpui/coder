package fuzzyfinder

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
	promptStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	itemStyle            = lipgloss.NewStyle().PaddingLeft(2)
	selectedItemStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).PaddingLeft(1)
	headerStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Underline(true)
	maxHeight, minHeight = 20, 5
	width                = 80
)

type Model struct {
	TextInput     textinput.Model
	allItems      []string
	filteredItems []string
	cursor        int
	Choice        string
	Quitting      bool
}

func New(items []string) Model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = width - 4
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle

	// Filter out empty lines from items
	var validItems []string
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			validItems = append(validItems, item)
		}
	}

	return Model{
		TextInput:     ti,
		allItems:      validItems,
		filteredItems: validItems,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Quitting = true
			return m, nil
		case tea.KeyEnter:
			if m.cursor < len(m.filteredItems) {
				m.Choice = m.filteredItems[m.cursor]
			}
			m.Quitting = true
			return m, nil
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(m.filteredItems)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	m.filterItems()
	return m, cmd
}

func (m *Model) filterItems() {
	query := strings.TrimSpace(m.TextInput.Value())
	if query == "" {
		m.filteredItems = m.allItems
		m.cursor = 0
		return
	}

	var newFiltered []string
	for _, item := range m.allItems {
		if fuzzyMatch(query, item) {
			newFiltered = append(newFiltered, item)
		}
	}
	m.filteredItems = newFiltered
	if m.cursor >= len(m.filteredItems) {
		m.cursor = 0
		if len(m.filteredItems) > 0 {
			m.cursor = len(m.filteredItems) - 1
		}
	}
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Find Command"))
	b.WriteString("\n")
	b.WriteString(m.TextInput.View())
	b.WriteString("\n\n")

	viewHeight := len(m.filteredItems)
	if viewHeight > maxHeight-4 { // 4 for header, input, and borders
		viewHeight = maxHeight - 4
	}
	if viewHeight < minHeight-4 {
		viewHeight = minHeight - 4
	}

	start := 0
	if m.cursor >= viewHeight {
		start = m.cursor - viewHeight + 1
	}

	end := start + viewHeight
	if end > len(m.filteredItems) {
		end = len(m.filteredItems)
	}

	for i := start; i < end; i++ {
		item := m.filteredItems[i]
		if i == m.cursor {
			b.WriteString(selectedItemStyle.Render("▸ " + item))
		} else {
			b.WriteString(itemStyle.Render(item))
		}
		b.WriteString("\n")
	}

	content := strings.TrimRight(b.String(), "\n")
	return containerStyle.Width(width).Render(content)
}

func fuzzyMatch(pattern string, text string) bool {
	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)
	pidx := 0
	for _, r := range text {
		if pidx < len(pattern) && rune(pattern[pidx]) == r {
			pidx++
		}
	}
	return pidx == len(pattern)
}
