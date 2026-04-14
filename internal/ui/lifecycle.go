package ui

import (
	"github.com/sokinpui/coder/internal/commands"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

		isCommand := strings.HasPrefix(m.Chat.TextArea.Value(), ":")
		key, isKey := msg.(tea.KeyMsg)
		if isCommand && m.Chat.TextArea.Value() != originalValue && isKey && key.Type != tea.KeyUp && key.Type != tea.KeyDown {
			m.Chat.CommandHistoryCursor = len(m.Chat.CommandHistory)
			m.Chat.CommandHistoryModified = ""
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
			var argPrefix string
			if len(parts) > 1 && !hasTrailingSpace {
				argPrefix = parts[len(parts)-1]
			}

			suggestions := commands.GetCommandArgumentSuggestions(cmdName, m.Session.GetConfig(), argPrefix)
			if suggestions != nil {
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

	if !m.Chat.IsCyclingCompletions {
		m.Chat.PaletteOffset = 0
	}
	return m
}

func (m Model) updateLayout() Model {
	maxHeight := m.Height / 4
	visibleLines := getVisibleLines(m.Chat.TextArea, m.Chat.TextArea.Width(), maxHeight+1)
	inputHeight := min(visibleLines+1, maxHeight)
	m.Chat.TextArea.SetHeight(max(1, inputHeight))

	var viewportHeight int
	if m.State == stateHistorySelect {
		viewportHeight = m.Height - lipgloss.Height(m.historyHeaderView()) - lipgloss.Height(m.StatusView()) - 1
	} else {
		viewportHeight = m.Height - m.Chat.TextArea.Height() - lipgloss.Height(m.StatusView()) - textAreaStyle.GetVerticalPadding() - 2
	}

	if viewportHeight < 0 {
		viewportHeight = 0
	}
	m.Chat.Viewport.Height = viewportHeight
	return m
}
