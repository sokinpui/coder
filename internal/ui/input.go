package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

func CalculateVisibleLines(ta textarea.Model, width int, maxLines int) int {
	if width <= 0 {
		return 1
	}

	visibleLineCount := 0
	for line := range strings.SplitSeq(ta.Value(), "\n") {
		lineWidth := lipgloss.Width(line)
		if lineWidth == 0 {
			visibleLineCount++
			continue
		}
		visibleLineCount += (lineWidth-1)/width + 1
		if maxLines > 0 && visibleLineCount > maxLines {
			return visibleLineCount
		}
	}
	return visibleLineCount
}
