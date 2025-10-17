package update

import (
	"coder/internal/config"
	"coder/internal/types"
	"fmt"
	"strings"
	"time"

	"coder/internal/utils"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) newSession() (Model, tea.Cmd) {
	// The session handles saving and clearing messages.
	// The UI just needs to reset its state.
	m.Session.AddMessages(types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage})
	dirMsg := utils.GetDirInfoContent()
	m.Session.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})

	// Reset UI and state flags.
	m.LastInteractionFailed = false
	m.LastRenderedAIPart = ""
	m.TextArea.Focus()
	m.Viewport.GotoTop()
	m.Viewport.SetContent(m.renderConversation())

	// Recalculate the token count for the base context.
	m.IsCountingTokens = true
	return m, countTokensCmd(m.Session.GetPrompt())
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.TextArea.Value()

	// don't send if the input is empty
	if strings.TrimSpace(input) == "" {
		return m, nil
	}

	if !strings.HasPrefix(input, ":") {
		// This is a prompt, apply debounce.
		if m.LastInteractionFailed {
			m.Session.RemoveLastInteraction()
			m.LastInteractionFailed = false
		}

		m.Session.AddMessages(types.Message{Type: types.UserMessage, Content: input})

		var cmds []tea.Cmd
		if !m.Session.IsTitleGenerated() {
			cmds = append(cmds, generateTitleCmd(m.Session, input))
		}

		m.State = stateGenPending
		m.TextArea.Blur()
		m.TextArea.Reset()
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()

		cmds = append(cmds, tea.Tick(1*time.Second, func(t time.Time) tea.Msg { return startGenerationMsg{} }), m.Spinner.Tick)
		return m, tea.Batch(cmds...)
	}

	// This is a command, handle as before.
	if len(m.CommandHistory) == 0 || m.CommandHistory[len(m.CommandHistory)-1] != input {
		m.CommandHistory = append(m.CommandHistory, input)
	}
	m.CommandHistoryCursor = len(m.CommandHistory)
	m.commandHistoryModified = ""
	event := m.Session.HandleInput(input)

	switch event.Type {
	case types.NoOp:
		return m, nil

	case types.MessagesUpdated:
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()
		if !m.PreserveInputOnSubmit {
			m.TextArea.Reset()
		}
		m.PreserveInputOnSubmit = false
		m.IsCountingTokens = true
		return m, countTokensCmd(m.Session.GetPrompt())

	case types.NewSessionStarted:
		return m.newSession()

	case types.GenerationStarted:
		m, cmd := m.startGeneration(event)
		return m, cmd

	case types.VisualModeStarted:
		return m.enterVisualMode(visualModeNone)

	case types.GenerateModeStarted:
		return m.enterVisualMode(visualModeGenerate)
	case types.EditModeStarted:
		return m.enterVisualMode(visualModeEdit)
	case types.BranchModeStarted:
		return m.enterVisualMode(visualModeBranch)
	case types.HistoryModeStarted:
		m.State = stateHistorySelect
		m.TextArea.Blur()
		return m, listHistoryCmd(m.Session.GetHistoryManager())
	}

	return m, nil
}

func (m Model) handleKeyPressIdle(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		if strings.HasPrefix(m.TextArea.Value(), ":") {
			if m.CommandHistoryCursor == len(m.CommandHistory) {
				m.commandHistoryModified = m.TextArea.Value()
			}

			if msg.Type == tea.KeyUp {
				if m.CommandHistoryCursor > 0 {
					m.CommandHistoryCursor--
					m.TextArea.SetValue(m.CommandHistory[m.CommandHistoryCursor])
					m.TextArea.CursorEnd()
				}
			} else { // KeyDown
				if m.CommandHistoryCursor < len(m.CommandHistory) {
					m.CommandHistoryCursor++
					if m.CommandHistoryCursor == len(m.CommandHistory) {
						m.TextArea.SetValue(m.commandHistoryModified)
					} else {
						m.TextArea.SetValue(m.CommandHistory[m.CommandHistoryCursor])
					}
					m.TextArea.CursorEnd()
				}
			}
			return m, nil, true
		}

	case tea.KeyCtrlC:
		if m.TextArea.Value() != "" {
			m.ClearedInputBuffer = m.TextArea.Value()
			m.TextArea.Reset()
			m.CtrlCPressed = false
			return m, nil, false // Allow layout recalculation in the same update cycle
		}
		if m.CtrlCPressed {
			m.Quitting = true
			return m, tea.Quit, true
		}
		m.CtrlCPressed = true
		return m, ctrlCTimeout(), true

	case tea.KeyCtrlZ:
		if m.ClearedInputBuffer != "" {
			m.TextArea.SetValue(m.ClearedInputBuffer)
			m.TextArea.CursorEnd()
			m.ClearedInputBuffer = ""
		}
		return m, nil, true

	case tea.KeyEscape:
		// Enter visual mode, preserving the text in the text area.
		model, cmd := m.enterVisualMode(visualModeNone)
		return model, cmd, true

	case tea.KeyTab, tea.KeyShiftTab:
		numCommands := len(m.PaletteFilteredCommands)
		numArgs := len(m.PaletteFilteredArguments)
		totalItems := numCommands + numArgs

		if !m.ShowPalette || totalItems == 0 {
			return m, nil, true
		}

		// On the first Tab press, we just want to complete with the current selection (cursor 0).
		// On subsequent Tab presses, we cycle.
		// `isCyclingCompletions` tracks if we are in a cycle.
		if !m.IsCyclingCompletions {
			// This is the first Tab/Shift+Tab press.
			if msg.Type == tea.KeyShiftTab {
				// If it's Shift+Tab, start from the end.
				m.PaletteCursor = totalItems - 1
			}
			// For a normal Tab, cursor is already 0, so we do nothing.
		} else {
			// We are already in a completion cycle.
			switch msg.Type {
			case tea.KeyTab:
				m.PaletteCursor = (m.PaletteCursor + 1) % totalItems
			case tea.KeyShiftTab:
				m.PaletteCursor--
				if m.PaletteCursor < 0 {
					m.PaletteCursor = totalItems - 1
				}
			}
		}
		m.IsCyclingCompletions = true

		var selectedItem string
		isArgument := false
		if m.PaletteCursor < numCommands {
			selectedItem = m.PaletteFilteredCommands[m.PaletteCursor]
		} else {
			selectedItem = m.PaletteFilteredArguments[m.PaletteCursor-numCommands]
			isArgument = true
		}

		val := m.TextArea.Value()
		parts := strings.Fields(val)

		if isArgument {
			var prefixParts []string
			if len(parts) > 0 && !strings.HasSuffix(val, " ") {
				prefixParts = parts[:len(parts)-1]
			} else {
				prefixParts = parts
			}
			m.TextArea.SetValue(strings.Join(append(prefixParts, selectedItem), " "))
		} else { // command/action
			m.TextArea.SetValue(selectedItem)
		}
		m.TextArea.CursorEnd()
		return m, nil, true

	case tea.KeyEnter:
		totalItems := len(m.PaletteFilteredCommands) + len(m.PaletteFilteredArguments)
		if m.ShowPalette && totalItems == 1 {
			var selectedItem string
			isArgument := false
			if len(m.PaletteFilteredCommands) == 1 {
				selectedItem = m.PaletteFilteredCommands[0]
			} else {
				selectedItem = m.PaletteFilteredArguments[0]
				isArgument = true
			}

			if isArgument {
				val := m.TextArea.Value()
				parts := strings.Fields(val)
				var prefixParts []string
				if len(parts) > 0 && !strings.HasSuffix(val, " ") {
					prefixParts = parts[:len(parts)-1]
				} else {
					prefixParts = parts
				}
				m.TextArea.SetValue(strings.Join(append(prefixParts, selectedItem), " "))
			} else {
				m.TextArea.SetValue(selectedItem)
			}
			m.TextArea.CursorEnd()
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}

		// Smart enter: submit if it's a command.
		if strings.HasPrefix(m.TextArea.Value(), ":") {
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}
		// Otherwise, fall through to let the textarea handle the newline.
		return m, nil, false

	case tea.KeyCtrlH:
		m.State = stateHistorySelect
		m.TextArea.Blur()
		return m, listHistoryCmd(m.Session.GetHistoryManager()), true

	case tea.KeyCtrlE:
		if m.TextArea.Focused() {
			return m, editInEditorCmd(m.TextArea.Value()), true
		}

	case tea.KeyCtrlJ:
		model, cmd := m.handleSubmit()
		return model, cmd, true

	case tea.KeyCtrlN:
		event := m.Session.HandleInput(":new")
		switch event.Type {
		case types.NewSessionStarted:
			newModel, cmd := m.newSession()
			return newModel, cmd, true
		}
		return m, nil, true

	case tea.KeyCtrlB:
		event := m.Session.HandleInput(":branch")
		switch event.Type {
		case types.BranchModeStarted:
			model, cmd := m.enterVisualMode(visualModeBranch)
			return model, cmd, true
		case types.MessagesUpdated:
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
		}
		return m, nil, true

	case tea.KeyCtrlF:
		var fzfInput strings.Builder

		// mode
		for _, mode := range config.AvailableAppModes {
			fzfInput.WriteString(fmt.Sprintf("mode: %s\n", mode))
		}

		// model
		for _, model := range config.AvailableModels {
			fzfInput.WriteString(fmt.Sprintf("model: %s\n", model))
		}

		return m, runFzfCmd(fzfInput.String()), true

	case tea.KeyCtrlA:
		// Equivalent to typing ":itf" and pressing enter.
		event := m.Session.HandleInput(":itf")
		switch event.Type {
		case types.MessagesUpdated:
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
		}
		return m, nil, true

	case tea.KeyCtrlV:
		return m, handlePasteCmd(), true
	}
	return m, nil, false
}
