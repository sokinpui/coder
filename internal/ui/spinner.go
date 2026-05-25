package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// typingSpinner is a custom spinner that mimics a "typing" animation.
var typingSpinner = spinner.Spinner{
	Frames: []string{
		"...", // Placeholder, will be populated by init function.
	},
	FPS: time.Second / 5, // 5 frames per second.
}

// init populates the frames for the typingSpinner with styled dots.
func init() {
	dot := "•"
	frames := []string{
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGreyDotStyle.Render(dot), lightGreyDotStyle.Render(dot), lightGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGreyDotStyle.Render(dot), greyDotStyle.Render(dot), lightGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGreyDotStyle.Render(dot), greyDotStyle.Render(dot), darkGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, darkGreyDotStyle.Render(dot), lightGreyDotStyle.Render(dot), greyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, greyDotStyle.Render(dot), darkGreyDotStyle.Render(dot), lightGreyDotStyle.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGreyDotStyle.Render(dot), greyDotStyle.Render(dot), darkGreyDotStyle.Render(dot)),
	}
	typingSpinner.Frames = frames
}

