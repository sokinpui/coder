package ui

import (
	"fmt"
	"strings"
)

func (m Model) historyView() string {
	if len(m.HistoryItems) == 0 {
		return "No history found."
	}

	var b strings.Builder
	b.WriteString("Select a conversation to load:\n\n")

	for i, item := range m.HistoryItems {
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
