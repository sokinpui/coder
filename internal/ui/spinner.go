package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

var (
	// ANSI color codes for different shades of grey.
	// These are approximations based on the user's description.
	lightGrey = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	grey      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	darkGrey  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
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
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGrey.Render(dot), lightGrey.Render(dot), lightGrey.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGrey.Render(dot), grey.Render(dot), lightGrey.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGrey.Render(dot), grey.Render(dot), darkGrey.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, darkGrey.Render(dot), lightGrey.Render(dot), grey.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, grey.Render(dot), darkGrey.Render(dot), lightGrey.Render(dot)),
		lipgloss.JoinHorizontal(lipgloss.Bottom, lightGrey.Render(dot), grey.Render(dot), darkGrey.Render(dot)),
	}
	typingSpinner.Frames = frames
}
