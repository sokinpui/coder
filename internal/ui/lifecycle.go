package ui

import (
	"coder/internal/commands"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateComponents handles updates for the textarea and viewport based on focus.
func (m Model) updateComponents(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	isRuneKey := false
	isViewportNavKey := false
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyRunes, tea.KeySpace:
			isRuneKey = true
		case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
			isViewportNavKey = true
		}
	}

	if m.TextArea.Focused() {
		originalValue := m.TextArea.Value()
		m.TextArea, cmd = m.TextArea.Update(msg)
		cmds = append(cmds, cmd)

		if m.TextArea.Value() != originalValue && strings.HasPrefix(m.TextArea.Value(), ":") {
			if key, ok := msg.(tea.KeyMsg); ok {
				if key.Type != tea.KeyUp && key.Type != tea.KeyDown {
					m.CommandHistoryCursor = len(m.CommandHistory)
					m.commandHistoryModified = ""
				}
			}
		}

		if !isRuneKey && !isViewportNavKey {
			m.Viewport, cmd = m.Viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		m.Viewport, cmd = m.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updatePalette updates the state of the command palette based on the textarea's content.
func (m Model) updatePalette() Model {
	if m.IsCyclingCompletions {
		return m
	}

	val := m.TextArea.Value()
	m.PaletteFilteredCommands = []string{}
	m.PaletteFilteredArguments = []string{}

	if m.State == stateIdle && strings.HasPrefix(val, ":") {
		parts := strings.Fields(val)
		hasTrailingSpace := strings.HasSuffix(val, " ")

		if len(parts) == 0 { // Just ":"
			parts = []string{":"}
		}

		if len(parts) == 1 && !hasTrailingSpace {
			prefix := strings.TrimPrefix(parts[0], ":")
			for _, c := range m.AvailableCommands {
				if strings.HasPrefix(c, prefix) {
					m.PaletteFilteredCommands = append(m.PaletteFilteredCommands, ":"+c)
				}
			}
		} else if len(parts) >= 1 {
			cmdName := strings.TrimPrefix(parts[0], ":")
			suggestions := commands.GetCommandArgumentSuggestions(cmdName, m.Session.GetConfig())
			if suggestions != nil {
				var argPrefix string
				if len(parts) > 1 && !hasTrailingSpace {
					argPrefix = parts[len(parts)-1]
				}

				for _, s := range suggestions {
					if strings.HasPrefix(s, argPrefix) {
						m.PaletteFilteredArguments = append(m.PaletteFilteredArguments, s)
					}
				}
			}
		}
	}

	totalItems := len(m.PaletteFilteredCommands) + len(m.PaletteFilteredArguments)
	m.ShowPalette = totalItems > 0

	if m.PaletteCursor >= totalItems {
		m.PaletteCursor = 0
	}
	return m
}

// updateLayout recalculates the size and position of UI elements.
func (m Model) updateLayout() Model {
	visibleLines := getVisibleLines(m.TextArea.Value(), m.TextArea.Width())
	inputHeight := min(visibleLines+1, m.Height/4)
	m.SearchInput.Width = m.Width - lipgloss.Width(m.SearchInput.Prompt) - 2
	m.TextArea.SetHeight(max(1, inputHeight))

	statusViewHeight := lipgloss.Height(m.StatusView())

	var currentInputHeight int
	var inputAreaVerticalPadding int
	if m.State == stateSearching {
		currentInputHeight = 1
		inputAreaVerticalPadding = 0
	} else if m.State != stateHistorySelect && m.State != stateSearchNav {
		currentInputHeight = m.TextArea.Height()
		inputAreaVerticalPadding = textAreaStyle.GetVerticalPadding()
	}

	viewportHeight := m.Height - currentInputHeight - statusViewHeight - inputAreaVerticalPadding - 2
	if viewportHeight < 0 {
		viewportHeight = 0
	}
	m.Viewport.Height = viewportHeight
	return m
}
