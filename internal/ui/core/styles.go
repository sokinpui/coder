package core

import "github.com/charmbracelet/lipgloss"

var (
	// Message Styles
	InitMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true).
				Padding(0, 1).
				Bold(true)

	DirectoryWelcomeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Italic(true).
				Padding(0, 1).
				Bold(true)

	UserInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("244")).
			Padding(0, 1)

	ImageMessageStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				Foreground(lipgloss.Color("244")).
				BorderForeground(lipgloss.Color("244")).
				Padding(0, 1)

	CommandInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)

	CommandResultStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1).
				BorderTop(false).
				BorderBottom(false).
				BorderRight(false)

	CommandErrorStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("9")).
				Foreground(lipgloss.Color("9")).
				Padding(0, 1)

	// Tool Messages Styles
	ToolCallStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("221")).
			Bold(true)

	ToolSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("78"))

	ToolErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	ToolMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	// Status Bar Styles
	StatusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	ModelInfoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	TokenCountStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))

	StatusBarMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51"))

	StatusBarTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("228")).
				Bold(true)

	AskingStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("228")).
				Italic(true)

	ThinkingStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")).
				Italic(true)

	GeneratingStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("51")).
				Italic(true)

	ThinkingTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)

	// Input Styles
	TextAreaContainerStyle = lipgloss.NewStyle().
				Padding(1, 2, 0, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	// Palette & Modal Styles
	PaletteContainerStyle = lipgloss.NewStyle().
				Padding(0, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	PaletteHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true).
				MarginBottom(1)

	PaletteItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	PaletteSelectedItemStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("51"))

	PaletteDescriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	TabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("228")).
			Bold(true).
			Padding(0, 1)

	SearchPlaceholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	// Spinner Dot Styles
	LightGreyDotStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	GreyDotStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	DarkGreyDotStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)
