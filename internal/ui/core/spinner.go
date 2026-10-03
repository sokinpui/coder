package core

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
