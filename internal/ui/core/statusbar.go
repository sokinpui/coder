package core

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderStatusBar(width int, titleLine string, leftStatus string, rightItems []string) string {
	var filteredRight []string
	for _, item := range rightItems {
		if strings.TrimSpace(item) != "" {
			filteredRight = append(filteredRight, item)
		}
	}
	rightStatus := strings.Join(filteredRight, " | ")

	var statusLine string
	if leftStatus != "" {
		spacing := max(width-lipgloss.Width(leftStatus)-lipgloss.Width(rightStatus), 1)
		statusLine = lipgloss.JoinHorizontal(lipgloss.Top, leftStatus, strings.Repeat(" ", spacing), rightStatus)
	} else {
		statusLine = rightStatus
	}

	if titleLine == "" {
		return statusLine
	}
	return lipgloss.JoinVertical(lipgloss.Left, titleLine, statusLine)
}
