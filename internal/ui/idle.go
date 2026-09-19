package ui

import (
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

func (m Model) handleEvent(event types.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case types.NoOp:
		return m, nil

	case types.MessagesUpdated:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m = m.updateLayout()
		return m, m.updateTokenCountCmd()

	case types.NewSessionStarted:
		return m.newSession(event.Mode)

	case types.GenerationStarted:
		return m.startGeneration(event)

	case types.TermExecutionStarted:
		cmdStr, _ := event.Data.(string)
		m.ActiveOverlay = overlayNone
		m.Chat.TextArea.Blur()
		return m, execTerminalCmd(cmdStr)

	case types.Quit:
		m.Quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) newSession(mode string) (Model, tea.Cmd) {
	oldSess := m.Session

	if mode == "" {
		mode = "coding"
	}

	newSess, err := coder.New(m.Session.GetConfig(), mode, m.Session.GetInstruction(), m.Session.GetContextFiles())
	if err != nil {
		log.Printf("Error creating new session: %v", err)
		return m, nil
	}

	m.Session = newSess
	m.ClearCache()
	m.addActiveSession(newSess)
	m.Session.AddMessages(types.Message{Type: types.InitMessage, Content: welcomeMessage})
	dirMsg := project.DirInfo()
	m.Session.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})

	m.State = stateIdle
	m.Chat.IsStreaming = false
	m.Chat.IsAIRendering = false
	m.Chat.PendingAIRender = false
	m.Chat.LastInteractionFailed = false
	m.Chat.TextArea.Focus()
	m.Chat.Viewport.GotoTop()
	m.Chat.Viewport.SetContent(m.renderConversation())

	return m, tea.Batch(loadInitialContextCmd(m.Session), saveConversationCmd(oldSess))
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.Chat.TextArea.Value()

	// don't send if the input is empty
	if strings.TrimSpace(input) == "" {
		return m, nil
	}

	if !strings.HasPrefix(input, "/") {
		m.Session.AddMessages(types.Message{Type: types.UserMessage, Content: input})
		m.Chat.ShowPalette = false

		var cmds []tea.Cmd
		if !m.Session.IsTitleGenerated() {
			cmds = append(cmds, generateTitleCmd(m.Session, input))
		}
		cmds = append(cmds, m.updateTokenCountCmd())

		event := m.Session.StartGeneration()
		switch event.Type {
		case types.GenerationStarted:
			newModel, genCmd := m.startGeneration(event)
			cmds = append(cmds, genCmd)
			return newModel, tea.Batch(cmds...)
		case types.MessagesUpdated:
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			m.State = stateIdle
			m.Chat.TextArea.Focus()
			cmds = append(cmds, textarea.Blink)
			return m, tea.Batch(cmds...)
		}
		return m, tea.Batch(cmds...)
	}

	m.Chat.ShowPalette = false

	if model, cmd, handled := m.handleUICommand(input); handled {
		return model, cmd
	}

	event := m.Session.HandleInput(input)

	shouldPreserve := m.Chat.PreserveInputOnSubmit
	m.Chat.PreserveInputOnSubmit = false

	model, cmd := m.handleEvent(event)
	if newModel, ok := model.(Model); ok {
		isCommand := strings.HasPrefix(input, "/")
		if event.Type == types.MessagesUpdated ||
			event.Type == types.NewSessionStarted ||
			(isCommand && event.Type != types.NoOp) {
			if !shouldPreserve {
				newModel.Chat.TextArea.Reset()
			}
		}
		return newModel, cmd
	}

	return model, cmd
}

func (m Model) handleUICommand(input string) (tea.Model, tea.Cmd, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(input, "/"))
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return m, nil, false
	}

	cmdName := parts[0]
	args := strings.Join(parts[1:], " ")

	switch cmdName {
	case "msg", "cards", "gen", "edit", "branch":
		m.Chat.TextArea.Reset()
		newModel, cmd := m.openAtomicMsgMode()
		return newModel, cmd, true

	case "history":
		m.Chat.TextArea.Reset()
		newModel, cmd := m.openHistorySelector(0)
		return newModel, cmd, true

	case "active":
		m.Chat.TextArea.Reset()
		newModel, cmd := m.openHistorySelector(1)
		return newModel, cmd, true

	case "model":
		if strings.TrimSpace(args) == "" {
			m.Chat.TextArea.Reset()
			newModel, cmd := m.openModelSelector("")
			return newModel, cmd, true
		}
		return m, nil, false

	case "exclude":
		if strings.TrimSpace(args) == "" {
			m.Chat.TextArea.Reset()
			files := m.Session.GetContextFiles()
			if len(files) == 0 {
				m.StatusBarMessage = "No project source files in context."
				return m, clearStatusBarCmd(), true
			}
			newModel, cmd := m.openFileListSelector("── Exclude Files ──", "Filter files to exclude...", files, func(mod Model, selected []string) (tea.Model, tea.Cmd) {
				cmdStr := "/exclude " + strings.Join(selected, " ")
				ev := mod.Session.HandleInput(cmdStr)
				return mod.handleEvent(ev)
			})
			return newModel, cmd, true
		}
		return m, nil, false

	case "help", "list":
		m.Chat.TextArea.Reset()
		newModel, cmd := m.showQuickView("/" + cmdName)
		return newModel, cmd, true

	case "config":
		if strings.TrimSpace(args) != "reload" {
			m.Chat.TextArea.Reset()
			newModel, cmd := m.showQuickView("/config")
			return newModel, cmd, true
		}
		return m, nil, false
	}

	return m, nil, false
}

func (m Model) handleKeyPressIdle(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	keyStr := msg.String()
	km := m.Session.GetConfig().Keymap

	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		isCommand := strings.HasPrefix(m.Chat.TextArea.Value(), "/")
		numCommands := len(m.Chat.PaletteFilteredCommands)
		numArgs := len(m.Chat.PaletteFilteredArguments)
		totalItems := numCommands + numArgs
		isPaletteActive := isCommand && m.Chat.ShowPalette && totalItems > 0

		if isPaletteActive {
			if msg.Type == tea.KeyUp {
				m.Chat.PaletteCursor = (m.Chat.PaletteCursor - 1 + totalItems) % totalItems
			} else {
				m.Chat.PaletteCursor = (m.Chat.PaletteCursor + 1) % totalItems
			}
			m.Chat.IsCyclingCompletions = true
			m = m.applyPaletteSelection()
			return m, nil, true
		}

		if msg.Type == tea.KeyDown {
			// If on the last line of the text area, move cursor to the end.
			if m.Chat.TextArea.Line() == m.Chat.TextArea.LineCount()-1 {
				m.Chat.TextArea.CursorEnd()
				return m, nil, true
			}
		}

	case tea.KeyCtrlC:
		if m.Chat.TextArea.Value() != "" {
			m.Chat.TextArea.Reset()
			m.Chat.CtrlCPressed = false
			return m, nil, false // Allow layout recalculation in the same update cycle
		}
		if m.Chat.CtrlCPressed {
			m.Quitting = true
			return m, tea.Quit, true
		}
		m.Chat.CtrlCPressed = true
		return m, ctrlCTimeout(), true

	case tea.KeyTab, tea.KeyShiftTab:
		isCommand := strings.HasPrefix(m.Chat.TextArea.Value(), "/")
		numCommands := len(m.Chat.PaletteFilteredCommands)
		numArgs := len(m.Chat.PaletteFilteredArguments)
		totalItems := numCommands + numArgs
		isPaletteActive := isCommand && m.Chat.ShowPalette && totalItems > 0

		switch {
		case isPaletteActive:
			if !m.Chat.IsCyclingCompletions {
				if msg.Type == tea.KeyShiftTab {
					m.Chat.PaletteCursor = totalItems - 1
				}
			} else {
				if msg.Type == tea.KeyTab {
					m.Chat.PaletteCursor = (m.Chat.PaletteCursor + 1) % totalItems
				} else {
					m.Chat.PaletteCursor = (m.Chat.PaletteCursor - 1 + totalItems) % totalItems
				}
			}
			m.Chat.IsCyclingCompletions = true
			m = m.applyPaletteSelection()
			return m, nil, true

		case msg.Type == tea.KeyTab:
			m.Chat.TextArea.InsertString("  ")
			m = m.updateLayout()
			return m, nil, true
		}
		return m, nil, true

	case tea.KeyEnter:
		totalItems := len(m.Chat.PaletteFilteredCommands) + len(m.Chat.PaletteFilteredArguments)
		if m.Chat.ShowPalette && totalItems == 1 {
			var selectedItem string
			isArgument := false
			if len(m.Chat.PaletteFilteredCommands) == 1 {
				selectedItem = m.Chat.PaletteFilteredCommands[0]
			} else {
				selectedItem = m.Chat.PaletteFilteredArguments[0]
				isArgument = true
			}

			if isArgument {
				val := m.Chat.TextArea.Value()
				parts := strings.Fields(val)
				var prefixParts []string
				if len(parts) > 0 && !strings.HasSuffix(val, " ") {
					prefixParts = parts[:len(parts)-1]
				} else {
					prefixParts = parts
				}
				itemToInsert := strings.TrimSuffix(selectedItem, "/")
				m.Chat.TextArea.SetValue(strings.Join(append(prefixParts, itemToInsert), " "))
			} else {
				m.Chat.TextArea.SetValue(strings.TrimSuffix(selectedItem, "/"))
			}
			m.Chat.TextArea.CursorEnd()
			m.Chat.ShowPalette = false
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}

		// Smart enter: submit if it's a command.
		if strings.HasPrefix(m.Chat.TextArea.Value(), "/") {
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}
		// Otherwise, fall through to let the textarea handle the newline.
		return m, nil, false

	}

	switch keyStr {
	case km.Msg, "esc":
		if strings.HasPrefix(m.Chat.TextArea.Value(), "/") {
			m.Chat.TextArea.Reset()
			m.Chat.CtrlCPressed = false
			return m, nil, false
		}
		model, cmd := m.openAtomicMsgMode()
		return model, cmd, true

	case km.History:
		newModel, cmd := m.openHistorySelector(0)
		return newModel, cmd, true

	case km.Editor:
		if m.Chat.TextArea.Focused() {
			return m, editInEditorCmd(m.Chat.TextArea.Value()), true
		}

	case km.Submit:
		model, cmd := m.handleSubmit()
		return model, cmd, true

	case km.New:
		event := m.Session.HandleShortcut("/new")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case km.Branch:
		newModel, cmd := m.openAtomicMsgMode()
		return newModel, cmd, true

	case km.Finder:
		files := m.Session.GetContextFiles()
		if len(files) == 0 {
			m.StatusBarMessage = "No project source files in context."
			return m, clearStatusBarCmd(), true
		}
		newModel, cmd := m.openFileListSelector("── Search Context Files ──", "Search context files...", files, func(mod Model, selected []string) (tea.Model, tea.Cmd) {
			return mod, openFilesInEditorCmd(selected)
		})
		return newModel, cmd, true

	case km.AddFile:
		cfg := m.Session.GetConfig()
		newModel, cmd := m.openFileListSelector("── Add Files to Context ──", "Search files/directories to add...", nil, func(mod Model, selected []string) (tea.Model, tea.Cmd) {
			cmdStr := "/file " + strings.Join(selected, " ")
			ev := mod.Session.HandleInput(cmdStr)
			return mod.handleEvent(ev)
		})
		newModel.Selector.IsLoading = true
		return newModel, tea.Batch(cmd, scanAddFilesCmd(cfg.Context.Exclusions), m.Chat.Spinner.Tick), true

	case km.ApplyITF:
		// Equivalent to typing "/itf" and pressing enter.
		event := m.Session.HandleInput("/itf")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case km.Paste:
		return m, handlePasteCmd(m.Session.GetConfig()), true
	}
	return m, nil, false
}

func (m Model) showQuickView(cmdName string) (tea.Model, tea.Cmd) {
	res, _, _ := commands.ProcessCommand(cmdName, m.Session)
	m.QuickView.SetMessages([]types.Message{
		{Type: types.CommandMessage, Content: cmdName},
		{Type: types.CommandResultMessage, Content: res.Payload},
	})
	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()
	m.ActiveOverlay = overlayQuickView
	m.Chat.TextArea.Blur()
	return m, m.updateTokenCountCmd()
}

func (m Model) applyPaletteSelection() Model {
	numCommands := len(m.Chat.PaletteFilteredCommands)

	maxPaletteItems := max(5, m.Height/4)
	if m.Chat.PaletteCursor < m.Chat.PaletteOffset {
		m.Chat.PaletteOffset = m.Chat.PaletteCursor
	} else if m.Chat.PaletteCursor >= m.Chat.PaletteOffset+maxPaletteItems {
		m.Chat.PaletteOffset = m.Chat.PaletteCursor - maxPaletteItems + 1
	}

	var selectedItem string
	isArgument := false
	if m.Chat.PaletteCursor < numCommands {
		selectedItem = m.Chat.PaletteFilteredCommands[m.Chat.PaletteCursor]
	} else {
		selectedItem = m.Chat.PaletteFilteredArguments[m.Chat.PaletteCursor-numCommands]
		isArgument = true
	}

	val := m.Chat.TextArea.Value()
	parts := strings.Fields(val)

	if isArgument {
		var prefixParts []string
		if len(parts) > 0 && !strings.HasSuffix(val, " ") {
			prefixParts = parts[:len(parts)-1]
		} else {
			prefixParts = parts
		}
		itemToInsert := strings.TrimSuffix(selectedItem, "/")
		m.Chat.TextArea.SetValue(strings.Join(append(prefixParts, itemToInsert), " "))
	} else {
		m.Chat.TextArea.SetValue(strings.TrimSuffix(selectedItem, "/"))
	}
	m = m.updateLayout()
	m.Chat.TextArea.CursorEnd()
	return m
}
