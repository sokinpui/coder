package ui

import (
	"fmt"
	"strings"
)

func (m Model) historyHeaderView() string {
	var b strings.Builder
	if m.IsHistorySearching {
		b.WriteString("Search History: ")
		b.WriteString(m.HistorySearchInput.View())
		b.WriteString("\n\n")
	} else {
		b.WriteString("Select a conversation to load (type / to search):\n\n")
	}
	return b.String()
}

func (m Model) historyListView() string {
	var b strings.Builder

	if len(m.FilteredHistoryItems) == 0 {
		b.WriteString("  No matching history found.")
		return b.String()
	}

	for i, item := range m.FilteredHistoryItems {
		title := item.Title
		date := fmt.Sprintf("(%s)", item.ModifiedAt.Format("2006-01-02 15:04"))
		if i == m.HistoryCussorPos {
			b.WriteString(paletteSelectedItemStyle.Render("▸  " + title))
			b.WriteString(paletteItemStyle.Render(" " + date))
		} else {
			b.WriteString(paletteItemStyle.Render("   " + title + " " + date))
		}
		b.WriteString("\n")
	}

	return b.String()
}
