package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

type PickerAction func(m Model, selected []string, primary string) (tea.Model, tea.Cmd)

type PickerModel struct {
	TextInput  textinput.Model
	AllItems   []string
	FoundItems []string
	Selected   map[string]struct{}
	Cursor     int
	Width      int
	Height     int
	OnSelect   PickerAction
}

func NewPicker() PickerModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	return PickerModel{
		TextInput: ti,
		Selected:  make(map[string]struct{}),
	}
}

func (m PickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *PickerModel) updateFoundItems() {
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

func (m PickerModel) getSelectedItems() []string {
	var selectedList []string
	for _, item := range m.AllItems {
		if _, ok := m.Selected[item]; ok {
			selectedList = append(selectedList, item)
		}
	}
	if len(selectedList) == 0 && len(m.FoundItems) > 0 && m.Cursor < len(m.FoundItems) {
		selectedList = append(selectedList, m.FoundItems[m.Cursor])
	}
	return selectedList
}

func (m PickerModel) getPrimaryItem() string {
	if len(m.FoundItems) > 0 && m.Cursor < len(m.FoundItems) {
		return m.FoundItems[m.Cursor]
	}
	return ""
}

func (m PickerModel) View() string {
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
		cursorPrefix := "  "
		if actualIndex == m.Cursor {
			cursorPrefix = "▸ "
		}
		checkPrefix := ""
		if _, isSelected := m.Selected[item]; isSelected {
			checkPrefix = "[✓] "
		}
		line := fmt.Sprintf("%s%s%s", cursorPrefix, checkPrefix, item)
		if actualIndex == m.Cursor || checkPrefix != "" {
			b.WriteString(paletteSelectedItemStyle.Render(line))
		} else {
			b.WriteString(paletteItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	return paletteContainerStyle.Width(m.Width).Render(b.String())
}

type PickerOverlay struct{}

func (p *PickerOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayPicker
}

func (p *PickerOverlay) View(main *Model) string {
	pickerWidth := main.Width / 2
	pickerHeight := main.Height / 2
	if pickerWidth < 60 {
		pickerWidth = 60
	}
	if pickerHeight < 10 {
		pickerHeight = 10
	}
	main.Picker.Width = pickerWidth
	main.Picker.Height = pickerHeight
	main.Picker.TextInput.Width = pickerWidth - 4

	pickerContent := main.Picker.View()
	if pickerContent == "" {
		return main.View()
	}

	return OverlayCenter(pickerContent, main.View())
}
