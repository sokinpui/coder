package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	Title          string
	Items          []SelectorItem
	FilteredItems  []SelectorItem
	Selected       map[string]struct{}
	Cursor         int
	Anchor         int
	IsSelecting    bool
	SearchInput    textinput.Model
	ShowSearch     bool
	IsSearching    bool
	Tabs           []string
	ActiveTab      int
	GGPressed      bool
	Width          int
	Height         int
	CustomWidth    int
	CustomHeight   int
	ShowPreview    bool
	PreviewTitle   string
	PreviewDescription string
	PreviewContent string
	PreviewOffset  int
	IsLoading      bool
	MultiSelect    bool

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
	var topTitle string

	if s.ShowPreview {
		if s.Title != "" {
			topTitle = PaletteHeaderStyle.Render(s.Title)
		}
		if s.ShowSearch {
			header.WriteString(s.SearchInput.View())
			header.WriteString("\n\n")
		}
	} else if len(s.Tabs) > 0 {
		if s.Title != "" {
			header.WriteString(PaletteHeaderStyle.Render(s.Title))
			header.WriteString("\n")
		}
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
		if s.ShowPreview {
			availHeight := max(6, s.Height-2)
			if topTitle != "" {
				availHeight = max(4, availHeight-lipgloss.Height(topTitle))
			}
			maxItems = max(3, availHeight-3)
		}
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

			desc := strings.TrimSpace(item.Description)
			prefix := cursorPrefix + checkPrefix + badgeStr
			prefixW := lipgloss.Width(prefix)
			contentW := max(5, itemWidth-prefixW)
			titleW := lipgloss.Width(item.Title)

			var line string
			if s.ShowPreview || desc == "" || contentW <= titleW+5 {
				title := item.Title
				if titleW > contentW {
					title = ansi.Truncate(title, contentW, "...")
				}
				if isCursor || isSelected {
					line = PaletteSelectedItemStyle.Render(prefix + title)
				} else {
					line = prefix + PaletteItemStyle.Render(title)
				}
			} else {
				title := item.Title
				availForDesc := contentW - titleW - 2
				descText := ansi.Truncate(desc, availForDesc, "...")

				if isCursor || isSelected {
					line = PaletteSelectedItemStyle.Render(prefix+title) + "  " + PaletteDescriptionStyle.Render(descText)
				} else {
					line = prefix + PaletteItemStyle.Render(title) + "  " + PaletteDescriptionStyle.Render(descText)
				}
			}

			listBuf.WriteString(line)
			listBuf.WriteString("\n")
		}
	}

	contentParts := []string{header.String(), strings.TrimRight(listBuf.String(), "\n")}

	if s.ShowPreview {
		return s.renderSplitView(topTitle, header.String(), listBuf.String())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	return PaletteContainerStyle.Width(s.Width).Render(content)
}

func (s *SelectorModel) renderSplitView(topTitle, leftHeaderStr, listStr string) string {
	innerWidth := max(30, s.Width-6)
	innerHeight := max(10, s.Height-2)
	if topTitle != "" {
		innerHeight = max(6, innerHeight-lipgloss.Height(topTitle))
	}
	leftWidth := (innerWidth * 45) / 100
	gapWidth := 2
	rightWidth := innerWidth - leftWidth - gapWidth

	contentParts := []string{leftHeaderStr, strings.TrimRight(listStr, "\n")}
	leftContent := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
	leftPane := lipgloss.NewStyle().Width(leftWidth).Height(innerHeight).Render(leftContent)

	previewInnerWidth := max(10, rightWidth-4)

	titleText := s.PreviewTitle
	descText := s.PreviewDescription
	if (titleText == "" || descText == "") && len(s.FilteredItems) > 0 && s.Cursor < len(s.FilteredItems) {
		item := s.FilteredItems[s.Cursor]
		if titleText == "" {
			titleText = item.Title
		}
		if descText == "" {
			descText = item.Description
		}
	}

	var descLines []string
	if titleText != "" {
		descLines = append(descLines, PreviewTitleStyle.MaxWidth(previewInnerWidth).Render(titleText))
	}
	if descText != "" {
		descLines = append(descLines, PreviewDescriptionStyle.MaxWidth(previewInnerWidth).Render(descText))
	}

	var rightPane string
	if len(descLines) == 0 {
		previewInnerHeight := max(4, innerHeight-2)
		rightPane = s.renderPreviewBox(previewInnerWidth, previewInnerHeight)
	} else {
		descContent := strings.Join(descLines, "\n")
		actualDescLines := len(strings.Split(descContent, "\n"))
		descContentHeight := min(actualDescLines, max(1, (innerHeight-6)/2))
		previewContentHeight := max(2, innerHeight-4-descContentHeight)

		descBox := PreviewBoxStyle.
			Width(previewInnerWidth).
			Height(descContentHeight).
			Render(descContent)

		previewBox := s.renderPreviewBox(previewInnerWidth, previewContentHeight)
		rightPane = lipgloss.JoinVertical(lipgloss.Left, descBox, previewBox)
	}

	combined := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, strings.Repeat(" ", gapWidth), rightPane)
	if topTitle != "" {
		combined = lipgloss.JoinVertical(lipgloss.Left, topTitle, combined)
	}
	return PaletteContainerStyle.Width(s.Width).Render(combined)
}

func (s *SelectorModel) renderPreviewBox(innerWidth, innerHeight int) string {
	var rightBuf strings.Builder
	if strings.TrimSpace(s.PreviewContent) == "" {
		rightBuf.WriteString(ToolMutedStyle.Render("No preview available."))
	} else {
		rendered := markdown.Render(s.PreviewContent, innerWidth)
		lines := strings.Split(rendered, "\n")
		bodyMaxLines := max(1, innerHeight)
		maxStart := max(0, len(lines)-bodyMaxLines)
		startLine := clamp(s.PreviewOffset, 0, maxStart)
		endLine := min(startLine+bodyMaxLines, len(lines))
		rightBuf.WriteString(strings.Join(lines[startLine:endLine], "\n"))
	}

	return PreviewBoxStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(rightBuf.String())
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
