package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type SelectorItem struct {
	ID          string
	Title       string
	Description string
	Badge       string
	SearchText  string
	Data        any
}

type SelectorModel struct {
	Title         string
	Items         []SelectorItem
	FilteredItems []SelectorItem
	Selected      map[string]struct{}
	Cursor        int
	Anchor        int
	IsSelecting   bool
	SearchInput   textinput.Model
	ShowSearch    bool
	IsSearching   bool
	Tabs          []string
	ActiveTab     int
	GGPressed     bool
	Width         int
	Height        int
	FooterHelp    string
	IsLoading     bool

	OnConfirm      func(m Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd)
	OnCancel       func(m Model) (tea.Model, tea.Cmd)
	OnTabChange    func(m Model, newTab int) (tea.Model, tea.Cmd)
	OnCursorChange func(m Model, current *SelectorItem) Model
	KeyHandler     func(m Model, key tea.KeyMsg) (tea.Model, tea.Cmd, bool)
}

func NewSelector() SelectorModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Prompt = ""
	ti.CharLimit = 156
	ti.PlaceholderStyle = searchPlaceholderStyle

	return SelectorModel{
		SearchInput: ti,
		Selected:    make(map[string]struct{}),
	}
}

func (s *SelectorModel) SetItems(items []SelectorItem) {
	s.Items = items
	s.UpdateFilter()
}

func (s *SelectorModel) UpdateFilter() {
	query := strings.TrimSpace(s.SearchInput.Value())
	if query == "" {
		s.FilteredItems = s.Items
	} else {
		targets := make([]string, len(s.Items))
		for i, item := range s.Items {
			if item.SearchText != "" {
				targets[i] = item.SearchText
			} else if item.Description != "" {
				targets[i] = item.Title + " " + item.Description
			} else {
				targets[i] = item.Title
			}
		}
		matches := fuzzy.Find(query, targets)
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].Index < matches[j].Index
		})
		s.FilteredItems = make([]SelectorItem, len(matches))
		for i, m := range matches {
			s.FilteredItems[i] = s.Items[m.Index]
		}
	}

	if s.Cursor >= len(s.FilteredItems) {
		s.Cursor = max(0, len(s.FilteredItems)-1)
	}
}

func (s *SelectorModel) GetSelectedItems() []SelectorItem {
	if s.IsSelecting && len(s.FilteredItems) > 0 {
		start := min(s.Anchor, s.Cursor)
		end := max(s.Anchor, s.Cursor)
		if start < 0 {
			start = 0
		}
		if end >= len(s.FilteredItems) {
			end = len(s.FilteredItems) - 1
		}
		return s.FilteredItems[start : end+1]
	}

	if len(s.Selected) > 0 {
		var res []SelectorItem
		for _, it := range s.Items {
			if _, ok := s.Selected[it.ID]; ok {
				res = append(res, it)
			}
		}
		return res
	}

	if len(s.FilteredItems) > 0 && s.Cursor < len(s.FilteredItems) {
		return []SelectorItem{s.FilteredItems[s.Cursor]}
	}
	return nil
}

func (s *SelectorModel) GetPrimaryItem() *SelectorItem {
	if len(s.FilteredItems) > 0 && s.Cursor < len(s.FilteredItems) {
		return &s.FilteredItems[s.Cursor]
	}
	return nil
}

func (s *SelectorModel) View(main *Model) string {
	var header strings.Builder

	if len(s.Tabs) > 0 {
		for i, tab := range s.Tabs {
			tabStr := fmt.Sprintf("[ %s ]", tab)
			if i == s.ActiveTab {
				header.WriteString(activeTabStyle.Render(tabStr))
			} else {
				header.WriteString(tabStyle.Render(tabStr))
			}
			if i < len(s.Tabs)-1 {
				header.WriteString("  ")
			}
		}
		if s.IsSearching {
			header.WriteString("   Search: " + s.SearchInput.View())
		}
		header.WriteString("\n")
	} else if s.ShowSearch {
		header.WriteString(s.SearchInput.View())
		header.WriteString("\n\n")
	} else if s.Title != "" {
		header.WriteString(paletteHeaderStyle.Render(s.Title))
		header.WriteString("\n")
	}

	var listBuf strings.Builder
	if s.IsLoading {
		spinnerView := "•"
		if main != nil {
			spinnerView = main.Chat.Spinner.View()
		}
		listBuf.WriteString(fmt.Sprintf("  %s Loading files...\n", spinnerView))
	} else if len(s.FilteredItems) == 0 {
		listBuf.WriteString("  No matching items.\n")
	} else {
		maxItems := max(5, s.Height-6)
		start := 0
		if s.Cursor >= maxItems {
			start = s.Cursor - maxItems + 1
		}
		end := min(start+maxItems, len(s.FilteredItems))

		selectedSet := make(map[string]struct{})
		if s.IsSelecting {
			for _, item := range s.GetSelectedItems() {
				selectedSet[item.ID] = struct{}{}
			}
		} else {
			selectedSet = s.Selected
		}

		itemWidth := max(20, s.Width-4)
		for i := start; i < end; i++ {
			item := s.FilteredItems[i]
			isCursor := i == s.Cursor
			_, isSelected := selectedSet[item.ID]

			cursorPrefix := "  "
			if isCursor {
				cursorPrefix = "▸ "
			}

			checkPrefix := ""
			if s.IsSelecting || len(s.Selected) > 0 {
				if isSelected {
					checkPrefix = "[✓] "
				} else {
					checkPrefix = "[ ] "
				}
			}

			badgeStr := ""
			if item.Badge != "" {
				badgeStr = item.Badge + " "
			}

			title := item.Title
			availableWidth := max(10, itemWidth-lipgloss.Width(badgeStr)-lipgloss.Width(checkPrefix)-lipgloss.Width(item.Description)-5)
			runes := []rune(title)
			if len(runes) > availableWidth {
				if availableWidth > 3 {
					title = string(runes[:availableWidth-3]) + "..."
				} else {
					title = string(runes[:availableWidth])
				}
			}

			line := fmt.Sprintf("%s%s%s%s%s", cursorPrefix, checkPrefix, badgeStr, title, item.Description)
			if isCursor || isSelected {
				listBuf.WriteString(paletteSelectedItemStyle.Width(itemWidth).Render(line))
			} else {
				listBuf.WriteString(paletteItemStyle.Width(itemWidth).Render(line))
			}
			listBuf.WriteString("\n")
		}
	}

	contentParts := []string{header.String(), strings.TrimRight(listBuf.String(), "\n")}
	if s.FooterHelp != "" {
		footer := paletteHeaderStyle.Render(s.FooterHelp)
		contentParts = append(contentParts, footer)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	return paletteContainerStyle.Width(s.Width).Render(content)
}

type SelectorOverlay struct{}

func (s *SelectorOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlaySelector
}

func (s *SelectorOverlay) View(main *Model) string {
	modalWidth := min(90, max(50, main.Width-4))
	modalHeight := min(30, max(12, main.Height-4))
	main.Selector.Width = modalWidth
	main.Selector.Height = modalHeight
	main.Selector.SearchInput.Width = modalWidth - 20

	content := main.Selector.View(main)
	if content == "" {
		return main.View()
	}
	return OverlayCenter(content, main.View())
}
