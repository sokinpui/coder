package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
	"github.com/sokinpui/coder/internal/ui/markdown"
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
	CustomWidth   int
	CustomHeight  int
	ShowPreview   bool
	PreviewTitle  string
	PreviewContent string
	PreviewOffset int
	FooterHelp    string
	IsLoading     bool
	MultiSelect   bool

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
	ti.PlaceholderStyle = SearchPlaceholderStyle

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

	if s.MultiSelect && len(s.Selected) > 0 {
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
				header.WriteString(ActiveTabStyle.Render(tabStr))
			} else {
				header.WriteString(TabStyle.Render(tabStr))
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
		if s.Title != "" {
			header.WriteString(PaletteHeaderStyle.Render(s.Title))
			header.WriteString("\n")
		}
		header.WriteString(s.SearchInput.View())
		header.WriteString("\n\n")
	} else if s.Title != "" {
		header.WriteString(PaletteHeaderStyle.Render(s.Title))
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
		} else if s.MultiSelect {
			selectedSet = s.Selected
		}

		itemWidth := max(20, s.Width-4)
		if s.ShowPreview {
			innerWidth := max(30, s.Width-6)
			leftWidth := (innerWidth * 45) / 100
			itemWidth = max(15, leftWidth-2)
		}
		for i := start; i < end; i++ {
			item := s.FilteredItems[i]
			isCursor := i == s.Cursor
			_, isSelected := selectedSet[item.ID]

			cursorPrefix := "  "
			if isCursor {
				cursorPrefix = "▸ "
			}

			checkPrefix := ""
			if s.IsSelecting || (s.MultiSelect && len(s.Selected) > 0) {
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
				listBuf.WriteString(PaletteSelectedItemStyle.Width(itemWidth).Render(line))
			} else {
				listBuf.WriteString(PaletteItemStyle.Width(itemWidth).Render(line))
			}
			listBuf.WriteString("\n")
		}
	}

	contentParts := []string{header.String(), strings.TrimRight(listBuf.String(), "\n")}
	if s.FooterHelp != "" {
		footer := PaletteHeaderStyle.Render(s.FooterHelp)
		contentParts = append(contentParts, footer)
	}

	if s.ShowPreview {
		return s.renderSplitView(header.String(), listBuf.String())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	return PaletteContainerStyle.Width(s.Width).Render(content)
}

func (s *SelectorModel) renderSplitView(headerStr, listStr string) string {
	innerWidth := max(30, s.Width-6)
	innerHeight := max(10, s.Height-2)
	leftWidth := (innerWidth * 45) / 100
	gapWidth := 2
	rightWidth := innerWidth - leftWidth - gapWidth

	contentParts := []string{headerStr, strings.TrimRight(listStr, "\n")}
	if s.FooterHelp != "" {
		contentParts = append(contentParts, PaletteHeaderStyle.Render(s.FooterHelp))
	}
	leftContent := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	leftPane := lipgloss.NewStyle().Width(leftWidth).Height(innerHeight).Render(leftContent)

	previewInnerWidth := max(10, rightWidth-4)
	previewInnerHeight := max(4, innerHeight-2)

	var rightBuf strings.Builder
	headerLines := 0
	if s.PreviewTitle != "" {
		rightBuf.WriteString(PreviewTitleStyle.MaxWidth(previewInnerWidth).Render(s.PreviewTitle))
		rightBuf.WriteString("\n\n")
		headerLines = 2
	}

	if strings.TrimSpace(s.PreviewContent) == "" {
		rightBuf.WriteString(ToolMutedStyle.Render("No preview available."))
	} else {
		rendered := markdown.Render(s.PreviewContent, previewInnerWidth)
		lines := strings.Split(rendered, "\n")
		bodyMaxLines := max(1, previewInnerHeight-headerLines)
		maxStart := max(0, len(lines)-bodyMaxLines)
		startLine := clamp(s.PreviewOffset, 0, maxStart)
		endLine := min(startLine+bodyMaxLines, len(lines))
		rightBuf.WriteString(strings.Join(lines[startLine:endLine], "\n"))
	}

	renderedPreview := PreviewBoxStyle.
		Width(previewInnerWidth).
		Height(previewInnerHeight).
		Render(rightBuf.String())

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, strings.Repeat(" ", gapWidth), renderedPreview)
	return PaletteContainerStyle.Width(s.Width).Render(combined)
}

type SelectorOverlay struct{}

func (s *SelectorOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlaySelector
}

func (s *SelectorOverlay) View(main *Model) string {
	maxWidth := max(40, main.Width-4)
	maxHeight := max(10, main.Height-4)

	modalWidth := min(90, maxWidth)
	modalHeight := min(30, maxHeight)
	if main.Selector.ShowPreview {
		modalWidth = max(60, min((main.Width*9)/10, maxWidth))
		modalHeight = max(18, min((main.Height*85)/100, maxHeight))
	}
	if main.Selector.CustomWidth > 0 {
		modalWidth = min(main.Selector.CustomWidth, maxWidth)
	}
	if main.Selector.CustomHeight > 0 {
		modalHeight = min(main.Selector.CustomHeight, maxHeight)
	}
	main.Selector.Width = modalWidth
	main.Selector.Height = modalHeight
	if main.Selector.ShowPreview {
		main.Selector.SearchInput.Width = max(10, (modalWidth*45/100)-12)
	} else {
		main.Selector.SearchInput.Width = modalWidth - 20
	}

	content := main.Selector.View(main)
	if content == "" {
		return main.View()
	}
	return OverlayCenter(content, main.View())
}
