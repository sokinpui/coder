package ui

import (
	"github.com/sokinpui/coder/internal/commands"
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

	if m.Chat.TextArea.Focused() {
		originalValue := m.Chat.TextArea.Value()
		m.Chat.TextArea, cmd = m.Chat.TextArea.Update(msg)
		cmds = append(cmds, cmd)

		if m.Chat.TextArea.Value() != originalValue && strings.HasPrefix(m.Chat.TextArea.Value(), ":") {
			if key, ok := msg.(tea.KeyMsg); ok {
				if key.Type != tea.KeyUp && key.Type != tea.KeyDown {
					m.Chat.CommandHistoryCursor = len(m.Chat.CommandHistory)
					m.Chat.CommandHistoryModified = ""
				}
			}
		}

		if !isRuneKey && !isViewportNavKey {
			m.Chat.Viewport, cmd = m.Chat.Viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		m.Chat.Viewport, cmd = m.Chat.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updatePalette updates the state of the command palette based on the textarea's content.
func (m Model) updatePalette() Model {
	if m.Chat.IsCyclingCompletions {
		return m
	}

	val := m.Chat.TextArea.Value()
	m.Chat.PaletteFilteredCommands = []string{}
	m.Chat.PaletteFilteredArguments = []string{}

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
					m.Chat.PaletteFilteredCommands = append(m.Chat.PaletteFilteredCommands, ":"+c)
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
						m.Chat.PaletteFilteredArguments = append(m.Chat.PaletteFilteredArguments, s)
					}
				}
			}
		}
	}

	totalItems := len(m.Chat.PaletteFilteredCommands) + len(m.Chat.PaletteFilteredArguments)
	m.Chat.ShowPalette = totalItems > 0

	if m.Chat.PaletteCursor >= totalItems {
		m.Chat.PaletteCursor = 0
	}
	return m
}

// updateLayout recalculates the size and position of UI elements.
func (m Model) updateLayout() Model {
	visibleLines := getVisibleLines(m.Chat.TextArea.Value(), m.Chat.TextArea.Width())
	inputHeight := min(visibleLines+1, m.Height/4)
	m.Chat.TextArea.SetHeight(max(1, inputHeight))

	statusViewHeight := lipgloss.Height(m.StatusView())

	var viewportHeight int
	if m.State == stateHistorySelect {
		headerHeight := lipgloss.Height(m.historyHeaderView())
		viewportHeight = m.Height - headerHeight - statusViewHeight - 1
	} else {
		viewportHeight = m.Height - m.Chat.TextArea.Height() - statusViewHeight - textAreaStyle.GetVerticalPadding() - 2
	}

	if viewportHeight < 0 {
		viewportHeight = 0
	}
	m.Chat.Viewport.Height = viewportHeight
	return m
}
