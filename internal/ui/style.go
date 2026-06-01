package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Message Styles
	initMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true).
				Padding(0, 1).
				Bold(true)
	directoryWelcomeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Italic(true).
				Padding(0, 1).
				Bold(true)

	// Status Bar Styles
	statusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
	modelInfoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))  // Blue
	tokenCountStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))  // Green

	// Palette Styles
	paletteContainerStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
	paletteHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true).
				MarginBottom(1)
	paletteItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
	paletteSelectedItemStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("51"))
	paletteDescriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	// Input Styles
	textAreaStyle = lipgloss.NewStyle().
			Padding(1, 2, 0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	userInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("244")).
			Padding(0, 1)
	imageMessageStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Foreground(lipgloss.Color("244")).
				BorderForeground(lipgloss.Color("244")).
				Padding(0, 1)
	commandInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)
	commandResultStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1).
				BorderTop(false).
				BorderBottom(false).
				BorderRight(false)
	commandErrorStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("9")). // Red
				Foreground(lipgloss.Color("9")).       // Red
				Padding(0, 1)

	// UI State Styles
	generatingStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")). // Cyan
				Italic(true)
	statusBarMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")) // Cyan
	statusBarTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("228")). // Yellow
				Bold(true)
	tabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)
	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("228")).
			Bold(true).
			Padding(0, 1)

	// Visual Mode Highlights & Bases
	visualCursorColor         = lipgloss.Color("51")  // Cyan
	visualSelectedColor       = lipgloss.Color("78")  // Green
	visualCursorSelectedColor = lipgloss.Color("228") // Yellow

	aiVisualBaseStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	commandResultVisualBaseStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("99")).
					Padding(0, 1)

	thinkingTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)

	// Spinner dot colors
	lightGreyDotStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	greyDotStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	darkGreyDotStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	// Placeholder / Disabled Styles
	disabledPlaceholderStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("240")).
					Italic(true)

	searchPlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

func applyHighlight(style lipgloss.Style, isCursorOn bool, isSelected bool) lipgloss.Style {
	if isCursorOn && isSelected {
		return style.Bold(true).BorderForeground(visualCursorSelectedColor)
	}
	if isCursorOn {
		return style.Bold(true).BorderForeground(visualCursorColor)
	}
	if isSelected {
		return style.BorderForeground(visualSelectedColor)
	}
	return style
}
