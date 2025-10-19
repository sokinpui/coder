package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Manager is the top-level model that manages the main view and the command palette overlay.
type Manager struct {
	Main     *Model
	Overlays []Overlay
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
	for _, o := range m.Overlays {
		if o.IsVisible(m.Main) {
			return o.View(m.Main)
		}
	}

	return m.Main.View()
}
