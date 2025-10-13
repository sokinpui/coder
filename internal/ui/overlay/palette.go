package overlay

import (
	"coder/internal/ui/update"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rmhubbert/bubbletea-overlay"
)

// PaletteOverlay implements the update.Overlay interface for the command palette.
type PaletteOverlay struct{}

// IsVisible checks if the command palette should be shown.
func (p *PaletteOverlay) IsVisible(main *update.Model) bool {
	return main.ShowPalette
}

// View renders the command palette overlay.
func (p *PaletteOverlay) View(main *update.Model) string {
	paletteContent := main.PaletteView()
	if paletteContent == "" {
		return main.View()
	}

	paletteModel := simpleModel{content: paletteContent}

	yOffset := (main.TextArea.Height() + lipgloss.Height(main.StatusView()) + 2) * -1

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
