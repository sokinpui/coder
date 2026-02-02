package ui

import (
	"fmt"
	"strings"
)

func (m Model) historyView() string {
	var b strings.Builder
	if m.IsHistorySearching {
		b.WriteString("Search History: ")
		b.WriteString(m.HistorySearchInput.View())
		b.WriteString("\n\n")
	} else {
		b.WriteString("Select a conversation to load (type / to search):\n\n")
	}

	if len(m.FilteredHistoryItems) == 0 {
		b.WriteString("  No matching history found.")
		return b.String()
	}

	for i, item := range m.FilteredHistoryItems {
		line := fmt.Sprintf("  %s (%s)", item.Title, item.ModifiedAt.Format("2006-01-02 15:04"))
		if i == m.HistoryCussorPos {
			b.WriteString(paletteSelectedItemStyle.Render("▸" + line))
		} else {
			b.WriteString(paletteItemStyle.Render(" " + line))
		}
		b.WriteString("\n")
	}

	return b.String()
}
