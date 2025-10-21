package ui

import (
	"strings"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rmhubbert/bubbletea-overlay"
)

// PaletteOverlay implements the Overlay interface for the command palette.
type PaletteOverlay struct{}

// IsVisible checks if the command palette should be shown.
func (p *PaletteOverlay) IsVisible(main *Model) bool {
	return main.ShowPalette
}

// View renders the command palette overlay.
func (p *PaletteOverlay) View(main *Model) string {
	paletteContent := main.PaletteView()
	if paletteContent == "" {
		return main.View()
	}

	paletteModel := simpleModel{content: paletteContent}

	yOffset := (main.TextArea.Height() + lipgloss.Height(main.StatusView()) + 1) * -1

	overlayModel := overlay.New(
		paletteModel,
		main,
		overlay.Left,
		overlay.Bottom,
		2,
		yOffset,
	)
	return overlayModel.View()
}

// simpleModel is a basic tea.Model for rendering a static string.
type simpleModel struct {
	content string
}

func (m simpleModel) Init() tea.Cmd                           { return nil }
func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m simpleModel) View() string                            { return m.content }

func (m Model) PaletteView() string {
	if !m.ShowPalette || (len(m.PaletteFilteredCommands) == 0 && len(m.PaletteFilteredArguments) == 0) {
		return ""
	}

	var b strings.Builder
	numCommands := len(m.PaletteFilteredCommands)

	if numCommands > 0 {
		b.WriteString(paletteHeaderStyle.Render("Commands"))
		b.WriteString("\n")
		for i, cmd := range m.PaletteFilteredCommands {
			cursorIndex := i
			if cursorIndex == m.PaletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + cmd))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + cmd))
			}
			b.WriteString("\n")
		}
	}

	if numCommands > 0 && len(m.PaletteFilteredArguments) > 0 {
		b.WriteString("\n")
	}

	if len(m.PaletteFilteredArguments) > 0 {
		b.WriteString(paletteHeaderStyle.Render("Arguments"))
		b.WriteString("\n")
		for i, arg := range m.PaletteFilteredArguments {
			cursorIndex := i + numCommands
			if cursorIndex == m.PaletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + arg))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + arg))
			}
			b.WriteString("\n")
		}
	}

	// Trim trailing newline
	content := strings.TrimRight(b.String(), "\n")

	return paletteContainerStyle.Render(content)
}
