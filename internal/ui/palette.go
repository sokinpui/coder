package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rmhubbert/bubbletea-overlay"
	"strings"
)

type PaletteOverlay struct{}

func (p *PaletteOverlay) IsVisible(main *Model) bool {
	return main.Chat.ShowPalette
}

func (p *PaletteOverlay) View(main *Model) string {
	paletteContent := main.PaletteView()
	if paletteContent == "" {
		return main.View()
	}

	paletteModel := simpleModel{content: paletteContent}

	yOffset := (main.Chat.TextArea.Height() + lipgloss.Height(main.StatusView()) + 1) * -1

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

type simpleModel struct {
	content string
}

func (m simpleModel) Init() tea.Cmd                           { return nil }
func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m simpleModel) View() string                            { return m.content }

func (m Model) PaletteView() string {
	if !m.Chat.ShowPalette || (len(m.Chat.PaletteFilteredCommands) == 0 && len(m.Chat.PaletteFilteredArguments) == 0) {
		return ""
	}

	maxItems := max(5, m.Height/4)
	var b strings.Builder
	numCommands := len(m.Chat.PaletteFilteredCommands)
	numArgs := len(m.Chat.PaletteFilteredArguments)
	total := numCommands + numArgs

	start := m.Chat.PaletteOffset
	end := start + maxItems
	if end > total {
		end = total
	}

	renderedCmds := 0
	if start < numCommands {
		b.WriteString(paletteHeaderStyle.Render("Commands"))
		b.WriteString("\n")

		secEnd := numCommands
		if end < numCommands {
			secEnd = end
		}

		for i := start; i < secEnd; i++ {
			item := m.Chat.PaletteFilteredCommands[i]
			if i == m.Chat.PaletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + item))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + item))
			}
			b.WriteString("\n")
			renderedCmds++
		}
	}

	if end > numCommands {
		if start <= numCommands {
			if renderedCmds > 0 {
				b.WriteString("\n")
			}
			b.WriteString(paletteHeaderStyle.Render("Arguments"))
			b.WriteString("\n")
		}

		secStart := numCommands
		if start > numCommands {
			secStart = start
		}

		for i := secStart; i < end; i++ {
			item := m.Chat.PaletteFilteredArguments[i-numCommands]
			if i == m.Chat.PaletteCursor {
				b.WriteString(paletteSelectedItemStyle.Render("▸ " + item))
			} else {
				b.WriteString(paletteItemStyle.Render("  " + item))
			}
			b.WriteString("\n")
		}
	}

	// Trim trailing newline
	content := strings.TrimRight(b.String(), "\n")

	return paletteContainerStyle.Render(content)
}
