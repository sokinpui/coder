package update

import (
	"coder/internal/ui/fuzzyfinder"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, loadInitialContextCmd(m.Session))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Reset cycling flag on any key press that is not Tab.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type != tea.KeyTab && key.Type != tea.KeyShiftTab {
		m.IsCyclingCompletions = false
	}

	if m.State == StateFzf {
		var cmd tea.Cmd
		newFzf, cmd := m.FuzzyFinder.Update(msg)
		if fzfModel, ok := newFzf.(fuzzyfinder.Model); ok {
			m.FuzzyFinder = fzfModel
		}

		if m.FuzzyFinder.Choice != "" {
			m.State = StateIdle
			m.TextArea.Focus()

			parts := strings.SplitN(m.FuzzyFinder.Choice, ": ", 2)
			var commandToRun string
			if len(parts) == 2 {
				commandToRun = fmt.Sprintf(":%s %s", parts[0], strings.TrimSpace(parts[1]))
			} else {
				commandToRun = ":" + m.FuzzyFinder.Choice
			}
			m.TextArea.SetValue(commandToRun)
			return m.handleSubmit()
		} else if m.FuzzyFinder.Quitting {
			m.State = StateIdle
			m.TextArea.Focus()
			return m, textarea.Blink
		}
		return m, cmd
	}

	// Handle state-specific messages and key presses.
	var handled bool
	var newModel tea.Model
	key, ok := msg.(tea.KeyMsg)
	if ok {
		newModel, cmd, handled = m.handleKeyPress(key)
		if handled {
			return newModel, cmd
		}
		m = newModel.(Model)
	} else {
		newModel, cmd, handled = m.handleMessage(msg)
		if handled {
			return newModel, cmd
		}
		m = newModel.(Model)
	}

	// Update sub-components like textarea and viewport.
	m, cmd = m.updateComponents(msg)
	cmds = append(cmds, cmd)

	// Update command palette based on textarea content.
	m = m.updatePalette()

	// Recalculate layout of all components.
	m = m.updateLayout()

	return m, tea.Batch(cmds...)
}
