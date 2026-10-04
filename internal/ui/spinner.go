package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

var TypingSpinner = spinner.Spinner{
	FPS: time.Second / 5,
}

func init() {
	dot := "•"
	TypingSpinner.Frames = []string{
		lipgloss.JoinHorizontal(lipgloss.Bottom, LightGreyDotStyle.Render(dot), LightGreyDotStyle.Render(dot), LightGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, LightGreyDotStyle.Render(dot), GreyDotStyle.Render(dot), LightGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, LightGreyDotStyle.Render(dot), GreyDotStyle.Render(dot), DarkGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, DarkGreyDotStyle.Render(dot), LightGreyDotStyle.Render(dot), GreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, GreyDotStyle.Render(dot), DarkGreyDotStyle.Render(dot), LightGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, LightGreyDotStyle.Render(dot), GreyDotStyle.Render(dot), DarkGreyDotStyle.Render(dot)),
	}
}

func RenderThinkingSpinner(text, spinnerView string) string {
	thinkingText := ThinkingTextStyle.Render(text)
	fullMessage := lipgloss.JoinHorizontal(lipgloss.Bottom, thinkingText, spinnerView)
	return lipgloss.NewStyle().Padding(0, 2).Render(fullMessage)
}
