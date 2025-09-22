package ui

import (
	"strings"

	"coder/internal/core"
	"coder/internal/session"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) newSession() (Model, tea.Cmd) {
	// The session handles saving and clearing messages.
	// The UI just needs to reset its state.
	m.session.AddMessage(core.Message{Type: core.InitMessage, Content: welcomeMessage})

	// Reset UI and state flags.
	m.lastInteractionFailed = false
	m.lastRenderedAIPart = ""
	m.textArea.Reset()
	m.textArea.Focus()
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderConversation())

	// Recalculate the token count for the base context.
	m.isCountingTokens = true
	return m, countTokensCmd(m.session.GetInitialPromptForTokenCount())
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.textArea.Value()

	// If the last interaction failed, the user is likely retrying.
	// Clear the previous failed attempt before submitting the new prompt.
	if !strings.HasPrefix(input, ":") && m.lastInteractionFailed {
		m.session.RemoveLastInteraction()
		m.lastInteractionFailed = false
	}

	var cmds []tea.Cmd
	shouldGenerateTitle := !strings.HasPrefix(input, ":") && !m.session.IsTitleGenerated()
	if shouldGenerateTitle {
		cmds = append(cmds, generateTitleCmd(m.session, input))
	}

	event := m.session.HandleInput(input)

	switch event.Type {
	case session.NoOp:
		return m, nil

	case session.MessagesUpdated:
		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()
		m.textArea.Reset()
		return m, tea.Batch(cmds...)

	case session.NewSessionStarted:
		return m.newSession()

	case session.GenerationStarted:
		m, cmd := m.startGeneration(event)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case session.VisualModeStarted:
		return m.enterVisualMode(visualModeNone)

	case session.GenerateModeStarted:
		return m.enterVisualMode(visualModeGenerate)
	case session.EditModeStarted:
		return m.enterVisualMode(visualModeEdit)
	case session.BranchModeStarted:
		return m.enterVisualMode(visualModeBranch)
	case session.HistoryModeStarted:
		m.state = stateHistorySelect
		m.textArea.Blur()
		return m, listHistoryCmd(m.session.GetHistoryManager())
	}

	return m, nil
}

func (m Model) handleKeyPressIdle(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.textArea.Value() != "" {
			m.clearedInputBuffer = m.textArea.Value()
			m.textArea.Reset()
			m.ctrlCPressed = false
			return m, nil, false // Allow layout recalculation in the same update cycle
		}
		if m.ctrlCPressed {
			m.quitting = true
			return m, tea.Quit, true
		}
		m.ctrlCPressed = true
		return m, ctrlCTimeout(), true

	case tea.KeyCtrlZ:
		if m.clearedInputBuffer != "" {
			m.textArea.SetValue(m.clearedInputBuffer)
			m.textArea.CursorEnd()
			m.clearedInputBuffer = ""
		}
		return m, nil, true

	case tea.KeyEscape:
		val := m.textArea.Value()
		if val == "" {
			model, cmd := m.enterVisualMode(visualModeNone)
			return model, cmd, true
		}

		if strings.HasPrefix(val, ":") {
			m.textArea.Reset()
		}
		// For normal prompts, do nothing.
		// In both cases, we've handled the event.
		return m, nil, true

	case tea.KeyTab, tea.KeyShiftTab:
		m.isCyclingCompletions = true

		numActions := len(m.paletteFilteredActions)
		numCommands := len(m.paletteFilteredCommands)
		numArgs := len(m.paletteFilteredArguments)
		totalItems := numActions + numCommands + numArgs

		if !m.showPalette || totalItems == 0 {
			return m, nil, true
		}

		if msg.Type == tea.KeyTab {
			m.paletteCursor = (m.paletteCursor + 1) % totalItems
		} else { // Shift+Tab
			m.paletteCursor--
			if m.paletteCursor < 0 {
				m.paletteCursor = totalItems - 1
			}
		}

		var selectedItem string
		isArgument := false
		if m.paletteCursor < numActions {
			selectedItem = m.paletteFilteredActions[m.paletteCursor]
		} else if m.paletteCursor < numActions+numCommands {
			selectedItem = m.paletteFilteredCommands[m.paletteCursor-numActions]
		} else {
			selectedItem = m.paletteFilteredArguments[m.paletteCursor-numActions-numCommands]
			isArgument = true
		}

		val := m.textArea.Value()
		parts := strings.Fields(val)

		if isArgument {
			var prefixParts []string
			if len(parts) > 0 && !strings.HasSuffix(val, " ") {
				prefixParts = parts[:len(parts)-1]
			} else {
				prefixParts = parts
			}
			m.textArea.SetValue(strings.Join(append(prefixParts, selectedItem), " "))
		} else { // command/action
			m.textArea.SetValue(selectedItem)
		}
		m.textArea.CursorEnd()
		return m, nil, true

	case tea.KeyEnter:
		totalItems := len(m.paletteFilteredActions) + len(m.paletteFilteredCommands) + len(m.paletteFilteredArguments)
		if m.showPalette && totalItems == 1 {
			var selectedItem string
			isArgument := false
			if len(m.paletteFilteredActions) == 1 {
				selectedItem = m.paletteFilteredActions[0]
			} else if len(m.paletteFilteredCommands) == 1 {
				selectedItem = m.paletteFilteredCommands[0]
			} else {
				selectedItem = m.paletteFilteredArguments[0]
				isArgument = true
			}

			if isArgument {
				val := m.textArea.Value()
				parts := strings.Fields(val)
				var prefixParts []string
				if len(parts) > 0 && !strings.HasSuffix(val, " ") {
					prefixParts = parts[:len(parts)-1]
				} else {
					prefixParts = parts
				}
				m.textArea.SetValue(strings.Join(append(prefixParts, selectedItem), " "))
			} else {
				m.textArea.SetValue(selectedItem)
			}
			m.textArea.CursorEnd()
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}

		// Smart enter: submit if it's a command.
		if strings.HasPrefix(m.textArea.Value(), ":") {
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}
		// Otherwise, fall through to let the textarea handle the newline.
		return m, nil, false

	case tea.KeyCtrlH:
		m.state = stateHistorySelect
		m.textArea.Blur()
		return m, listHistoryCmd(m.session.GetHistoryManager()), true

	case tea.KeyCtrlE:
		if m.textArea.Focused() {
			return m, editInEditorCmd(m.textArea.Value()), true
		}

	case tea.KeyCtrlJ:
		model, cmd := m.handleSubmit()
		return model, cmd, true

	case tea.KeyCtrlN:
		event := m.session.HandleInput(":new")
		if event.Type == session.NewSessionStarted {
			newModel, cmd := m.newSession()
			return newModel, cmd, true
		}
		return m, nil, true

	case tea.KeyCtrlB:
		event := m.session.HandleInput(":branch")
		if event.Type == session.BranchModeStarted {
			model, cmd := m.enterVisualMode(visualModeBranch)
			return model, cmd, true
		} else if event.Type == session.MessagesUpdated {
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
		}
		return m, nil, true

	case tea.KeyCtrlA:
		// Equivalent to typing ":itf" and pressing enter.
		event := m.session.HandleInput(":itf")
		if event.Type == session.MessagesUpdated {
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
		}
		return m, nil, true
	}
	return m, nil, false
}
