package update

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rmhubbert/bubbletea-overlay"
)

// Manager is the top-level model that manages the main view and the command palette overlay.
type Manager struct {
	Main *Model
}

func NewManager(main *Model) *Manager {
	return &Manager{Main: main}
}

func (m *Manager) Init() tea.Cmd {
	return m.Main.Init()
}

func (m *Manager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	newMain, cmd := m.Main.Update(msg)
	mainModel, ok := newMain.(Model)
	if ok {
		m.Main = &mainModel
	}
	return m, cmd
}

func (m *Manager) View() string {
	if m.Main.ShowPalette {
		paletteContent := m.Main.paletteView()
		if paletteContent == "" {
			return m.Main.View()
		}

		paletteModel := simpleModel{content: paletteContent}

		yOffset := (m.Main.TextArea.Height() + lipgloss.Height(m.Main.statusView()) + 1) * -1

		overlayModel := overlay.New(
			paletteModel,
			m.Main,
			overlay.Left,
			overlay.Bottom,
			2,
			yOffset,
		)
		return overlayModel.View()
	}

	return m.Main.View()
}

// simpleModel is a basic tea.Model for rendering a static string.
type simpleModel struct {
	content string
}

func (m simpleModel) Init() tea.Cmd                           { return nil }
func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m simpleModel) View() string                            { return m.content }
