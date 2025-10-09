package update

import (
	"strings"
	"time"

	"coder/internal/core"
	"coder/internal/utils"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) newSession() (Model, tea.Cmd) {
	// The session handles saving and clearing messages.
	// The UI just needs to reset its state.
	m.Session.AddMessage(core.Message{Type: core.InitMessage, Content: welcomeMessage})
	dirMsg := utils.GetDirInfoContent()
	m.Session.AddMessage(core.Message{Type: core.DirectoryMessage, Content: dirMsg})

	// Reset UI and state flags.
	m.LastInteractionFailed = false
	m.LastRenderedAIPart = ""
	m.TextArea.Reset()
	m.TextArea.Focus()
	m.Viewport.GotoTop()
	m.Viewport.SetContent(m.renderConversation())

	// Recalculate the token count for the base context.
	m.IsCountingTokens = true
	return m, countTokensCmd(m.Session.GetInitialPromptForTokenCount())
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.TextArea.Value()

	if !strings.HasPrefix(input, ":") {
		// This is a prompt, apply debounce.
		if m.LastInteractionFailed {
			m.Session.RemoveLastInteraction()
			m.LastInteractionFailed = false
		}

		m.Session.AddMessage(core.Message{Type: core.UserMessage, Content: input})

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
	event := m.Session.HandleInput(input)

	switch event.Type {
	case core.NoOp:
		return m, nil

	case core.MessagesUpdated:
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()
		m.TextArea.Reset()
		m.IsCountingTokens = true
		return m, countTokensCmd(m.Session.GetPromptForTokenCount())

	case core.NewSessionStarted:
		return m.newSession()

	case core.GenerationStarted:
		m, cmd := m.startGeneration(event)
		return m, cmd

	case core.VisualModeStarted:
		return m.enterVisualMode(visualModeNone)

	case core.GenerateModeStarted:
		return m.enterVisualMode(visualModeGenerate)
	case core.EditModeStarted:
		return m.enterVisualMode(visualModeEdit)
	case core.BranchModeStarted:
		return m.enterVisualMode(visualModeBranch)
	case core.HistoryModeStarted:
		m.State = stateHistorySelect
		m.TextArea.Blur()
		return m, listHistoryCmd(m.Session.GetHistoryManager())
	case core.FzfModeStarted:
		fzfInput, ok := event.Data.(string)
		if !ok {
			return m, nil
		}
		return m, runFzfCmd(fzfInput)
	}

	return m, nil
}

func (m Model) handleKeyPressIdle(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
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
		val := m.TextArea.Value()
		if val == "" {
			model, cmd := m.enterVisualMode(visualModeNone)
			return model, cmd, true
		}

		if strings.HasPrefix(val, ":") {
			m.TextArea.Reset()
		}
		// For normal prompts, do nothing.
		// In both cases, we've handled the event.
		return m, nil, true

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
			if msg.Type == tea.KeyTab {
				m.PaletteCursor = (m.PaletteCursor + 1) % totalItems
			} else { // Shift+Tab
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
		if event.Type == core.NewSessionStarted {
			newModel, cmd := m.newSession()
			return newModel, cmd, true
		}
		return m, nil, true

	case tea.KeyCtrlB:
		event := m.Session.HandleInput(":branch")
		switch event.Type {
		case core.BranchModeStarted:
			model, cmd := m.enterVisualMode(visualModeBranch)
			return model, cmd, true
		case core.MessagesUpdated:
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
		}
		return m, nil, true

	case tea.KeyCtrlF:
		event := m.Session.HandleInput(":fzf")
		switch event.Type {
		case core.FzfModeStarted:
			return m, runFzfCmd(event.Data.(string)), true
		}
		return m, nil, true

	case tea.KeyCtrlA:
		// Equivalent to typing ":itf" and pressing enter.
		event := m.Session.HandleInput(":itf")
		if event.Type == core.MessagesUpdated {
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
		}
		return m, nil, true
	}
	return m, nil, false
}
