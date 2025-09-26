package ui

import (
	"coder/internal/core"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, loadInitialContextCmd(m.session))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Reset cycling flag on any key press that is not Tab.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type != tea.KeyTab && key.Type != tea.KeyShiftTab {
		m.isCyclingCompletions = false
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		var handled bool
		var newModel tea.Model
		newModel, cmd, handled = m.handleKeyPress(key)
		if handled {
			return newModel, cmd
		}
		m = newModel.(Model)
	} else {
		var handled bool
		var newModel tea.Model
		newModel, cmd, handled = m.handleMessage(msg)
		if handled {
			return newModel, cmd
		}
		m = newModel.(Model)
	}

	// Handle updates for textarea and viewport based on focus.
	isRuneKey := false
	isViewportNavKey := false
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyRunes, tea.KeySpace:
			isRuneKey = true
		case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight,
			tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
			isViewportNavKey = true
		}
	}

	// When the textarea is focused, it gets all messages.
	// The viewport only gets messages that are not character runes.
	if m.textArea.Focused() {
		m.textArea, cmd = m.textArea.Update(msg)
		cmds = append(cmds, cmd)

		// Don't pass navigation keys to viewport when textarea is focused
		if !isRuneKey && !isViewportNavKey {
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		// When the textarea is not focused (e.g., during generation),
		// the viewport gets all messages.
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	visibleLines := calculateVisibleLines(m.textArea.Value(), m.textArea.Width())
	inputHeight := min(visibleLines+1, m.height/4)
	m.textArea.SetHeight(max(1, inputHeight))

	// After textarea update, check for palette
	if !m.isCyclingCompletions {
		val := m.textArea.Value()
		m.paletteFilteredCommands = []string{}
		m.paletteFilteredArguments = []string{}

		if m.state == stateIdle && strings.HasPrefix(val, ":") {
			parts := strings.Fields(val)
			hasTrailingSpace := strings.HasSuffix(val, " ")

			if len(parts) == 0 { // Just ":"
				parts = []string{":"}
			}

			if len(parts) == 1 && !hasTrailingSpace {
				// Command/Action completion mode
				prefix := strings.TrimPrefix(parts[0], ":")
				for _, c := range m.availableCommands {
					if strings.HasPrefix(c, prefix) {
						m.paletteFilteredCommands = append(m.paletteFilteredCommands, ":"+c)
					}
				}
			} else if len(parts) >= 1 {
				// Argument completion mode
				cmdName := strings.TrimPrefix(parts[0], ":")
				suggestions := core.GetCommandArgumentSuggestions(cmdName, m.session.GetConfig())
				if suggestions != nil {
					var argPrefix string
					if len(parts) > 1 && !hasTrailingSpace {
						argPrefix = parts[len(parts)-1]
					}

					for _, s := range suggestions {
						if strings.HasPrefix(s, argPrefix) {
							m.paletteFilteredArguments = append(m.paletteFilteredArguments, s)
						}
					}
				}
			}
		}

		totalItems := len(m.paletteFilteredCommands) + len(m.paletteFilteredArguments)
		m.showPalette = totalItems > 0

		if m.paletteCursor >= totalItems {
			m.paletteCursor = 0
		}
	}

	statusViewHeight := lipgloss.Height(m.statusView())

	paletteHeight := 0
	if m.showPalette {
		// We need a view to calculate its height.
		// This is a bit inefficient but necessary with lipgloss.
		paletteHeight = lipgloss.Height(m.paletteView())
	}

	viewportHeight := m.height - m.textArea.Height() - statusViewHeight - paletteHeight - textAreaStyle.GetVerticalPadding() - 2
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Height = viewportHeight

	return m, tea.Batch(cmds...)
}
