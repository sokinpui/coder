package ui

import (
	"context"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/engine"
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
		return m, tea.Batch(m.updateTokenCountCmd(), m.renderUncachedCmd())

	case types.NewSessionStarted:
		return m.newSession(event.Mode)

	case types.GenerationStarted:
		return m, nil

	case types.ShellExecutionStarted:
		cmdStr, _ := event.Data.(string)
		m.ActiveOverlay = overlayNone
		m.Chat.TextArea.Blur()
		return m, execShellCmd(cmdStr)

	case types.Quit:
		m.Quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) newSession(mode string) (Model, tea.Cmd) {
	oldSess := m.Session

	newSess, err := m.Session.CreateNew(mode)
	if err != nil {
		return m, nil
	}

	m.ActiveOverlay = overlayNone
	m.Session = newSess
	m.ClearCache()
	m.addActiveSession(newSess)
	m.AvailableCommands = newSess.GetSupportedCommands()
	m.CommandDescriptions = newSess.GetCommandDescriptions()
	sort.Strings(m.AvailableCommands)
	m.Session.AddMessages(types.Message{Type: types.InitMessage, Content: welcomeMessage})
	if dirMsg := project.DirInfo(); dirMsg != "" {
		m.Session.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})
	}

	m.State = stateIdle
	m.Chat.IsStreaming = false
	m.Chat.IsAIRendering = false
	m.Chat.PendingAIRender = false
	m.Chat.LastInteractionFailed = false
	m.Chat.TextArea.Reset()
	m.Chat.ShowPalette = false
	m.Chat.PaletteFilteredCommands = nil
	m.Chat.PaletteFilteredArguments = nil
	m.Chat.PaletteCursor = 0
	m.Chat.PaletteOffset = 0
	m.Chat.TextArea.Focus()
	m.Chat.Viewport.GotoTop()
	m.Chat.Viewport.SetContent(m.renderConversation())
	m = m.updateLayout()

	return m, tea.Batch(loadInitialContextCmd(m.Session), saveConversationCmd(oldSess), m.renderUncachedCmd())
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.Chat.TextArea.Value()

	// don't send if the input is empty
	if strings.TrimSpace(input) == "" {
		return m, nil
	}

	if !commands.IsCommand(input) {
		m.Chat.ShowPalette = false

		var cmds []tea.Cmd
		if !m.Session.IsTitleGenerated() {
			cmds = append(cmds, generateTitleCmd(m.Session, input))
		}
		cmds = append(cmds, m.updateTokenCountCmd())

		eventChan, err := m.Session.Submit(context.Background(), input)
		if err != nil {
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: err.Error(),
			})
			return m, tea.Batch(cmds...)
		}

		newModel, genCmd := m.startGenerationEvents(eventChan)
		cmds = append(cmds, genCmd)
		return newModel, tea.Batch(cmds...)
	}

	m.Chat.ShowPalette = false

	if model, cmd, handled := m.handleUICommand(input); handled {
		return model, cmd
	}

	cmdOut, _ := m.Session.ExecuteCommand(input)

	shouldPreserve := m.Chat.PreserveInputOnSubmit
	m.Chat.PreserveInputOnSubmit = false

	switch cmdOut.Type {
	case types.Quit:
		m.Quitting = true
		return m, tea.Quit
	case types.NewSessionStarted:
		return m.newSession(cmdOut.Mode)
	case types.ShellExecutionStarted:
		m.ActiveOverlay = overlayNone
		m.Chat.TextArea.Blur()
		return m, execShellCmd(cmdOut.Payload)
	case types.NoOp:
		return m, nil
	}

	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()
	m = m.updateLayout()
	if !shouldPreserve {
		m.Chat.TextArea.Reset()
	}

	return m, tea.Batch(m.updateTokenCountCmd(), m.renderUncachedCmd())
}

func (m Model) handleUICommand(input string) (tea.Model, tea.Cmd, bool) {
	cmdName, args, isCmd := commands.ParseCommand(input)
	if !isCmd || cmdName == "" {
		return m, nil, false
	}

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

	case "skills":
		m.Chat.TextArea.Reset()
		newModel, cmd := m.openSkillsSelector()
		return newModel, cmd, true

	case "mode":
		if strings.TrimSpace(args) == "" {
			if m.Session.HasChatHistory() {
				m.StatusBarMessage = "Cannot switch mode in a non-empty session. Use /new [mode]."
				return m, clearStatusBarCmd(), true
			}
			m.Chat.TextArea.Reset()
			newModel, cmd := m.openModeSelector()
			return newModel, cmd, true
		}
		return m, nil, false

	case "model":
		if strings.TrimSpace(args) == "" {
			m.Chat.TextArea.Reset()
			newModel, cmd := m.openModelSelector("")
			return newModel, cmd, true
		}
		return m, nil, false

	case "reasoning", "thinking":
		if !m.Session.Capabilities().Has(engine.CapReasoningSwitch) {
			return m, nil, false
		}
		if strings.TrimSpace(args) == "" {
			m.Chat.TextArea.Reset()
			newModel, cmd := m.openReasoningSelector()
			return newModel, cmd, true
		}
		return m, nil, false

	case "exclude":
		if !m.Session.Capabilities().Has(engine.CapContextFiles) {
			return m, nil, false
		}
		ctxCtrl, ok := m.Session.(engine.ContextController)
		if !ok {
			return m, nil, false
		}
		if strings.TrimSpace(args) == "" {
			m.Chat.TextArea.Reset()
			files := ctxCtrl.GetContextFiles()
			if len(files) == 0 {
				m.StatusBarMessage = "No project source files in context."
				return m, clearStatusBarCmd(), true
			}
			newModel, cmd := m.openFileListSelector("── Exclude Files ──", "Filter files to exclude...", files, func(mod Model, selected []string) (tea.Model, tea.Cmd) {
				cmdStr := "/exclude " + strings.Join(selected, " ")
				cmdOut, _ := mod.Session.ExecuteCommand(cmdStr)
				return mod.handleEvent(types.Event{Type: cmdOut.Type, Data: cmdOut.Payload})
			})
			return newModel, cmd, true
		}
		return m, nil, false

	case "list":
		if !m.Session.Capabilities().Has(engine.CapContextFiles) {
			return m, nil, false
		}
		m.Chat.TextArea.Reset()
		newModel, cmd := m.showQuickView("/list")
		return newModel, cmd, true

	case "help":
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
	km := m.Keymap()

	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		numCommands := len(m.Chat.PaletteFilteredCommands)
		numArgs := len(m.Chat.PaletteFilteredArguments)
		totalItems := numCommands + numArgs
		isPaletteActive := m.Chat.ShowPalette && totalItems > 0

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
		if m.Chat.CtrlCPressed {
			m.Quitting = true
			return m, tea.Quit, true
		}
		m.Chat.CtrlCPressed = true
		return m, ctrlCTimeout(), true

	case tea.KeyTab, tea.KeyShiftTab:
		numCommands := len(m.Chat.PaletteFilteredCommands)
		numArgs := len(m.Chat.PaletteFilteredArguments)
		totalItems := numCommands + numArgs
		isPaletteActive := m.Chat.ShowPalette && totalItems > 0

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
					if len(prefixParts) == 0 {
						prefixParts = determinePrefix(parts[0])
					}
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
		if commands.IsCommand(m.Chat.TextArea.Value()) {
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}
		// Otherwise, fall through to let the textarea handle the newline.
		return m, nil, false

	}

	switch keyStr {
	case km.Msg, "esc":
		if commands.IsCommand(m.Chat.TextArea.Value()) {
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
		model, cmd := m.newSession("")
		return model, cmd, true

	case km.Branch:
		newModel, cmd := m.openAtomicMsgMode()
		return newModel, cmd, true

	case km.Finder:
		if !m.Session.Capabilities().Has(engine.CapContextFiles) {
			return m, nil, false
		}
		ctxCtrl, ok := m.Session.(engine.ContextController)
		if !ok {
			return m, nil, false
		}
		files := ctxCtrl.GetContextFiles()
		if len(files) == 0 {
			m.StatusBarMessage = "No project source files in context."
			return m, clearStatusBarCmd(), true
		}
		newModel, cmd := m.openFileListSelector("── Search Context Files ──", "Search context files...", files, func(mod Model, selected []string) (tea.Model, tea.Cmd) {
			return mod, openFilesInEditorCmd(selected)
		})
		return newModel, cmd, true

	case km.AddFile:
		if !m.Session.Capabilities().Has(engine.CapContextFiles) {
			return m, nil, false
		}
		cfg := m.Session.GetConfig()
		newModel, cmd := m.openFileListSelector("── Add Files to Context ──", "Search files/directories to add...", nil, func(mod Model, selected []string) (tea.Model, tea.Cmd) {
			cmdStr := "/file " + strings.Join(selected, " ")
			cmdOut, _ := mod.Session.ExecuteCommand(cmdStr)
			return mod.handleEvent(types.Event{Type: cmdOut.Type, Data: cmdOut.Payload})
		})
		newModel.Selector.IsLoading = true
		return newModel, tea.Batch(cmd, scanAddFilesCmd(cfg.Coder.Context.Exclusions), m.Chat.Spinner.Tick), true

	case km.ApplyITF:
		if !m.Session.Capabilities().Has(engine.CapITF) {
			return m, nil, false
		}
		cmdOut, _ := m.Session.ExecuteCommand("/itf")
		model, cmd := m.handleEvent(types.Event{Type: cmdOut.Type, Data: cmdOut.Payload})
		return model, cmd, true

	case km.Paste:
		return m, HandlePasteCmd(m.Session.GetConfig()), true
	}
	return m, nil, false
}

func (m Model) showQuickView(cmdName string) (tea.Model, tea.Cmd) {
	ctrl, ok := m.Session.(commands.SessionController)
	if !ok {
		return m, nil
	}

	res, _, _ := commands.ProcessCommand(cmdName, ctrl)
	quickViewWidth := m.Width * 3 / 4
	quickViewHeight := m.Height * 3 / 4
	m.QuickView.Viewport.Width = quickViewWidth - PaletteContainerStyle.GetHorizontalFrameSize()
	m.QuickView.Viewport.Height = quickViewHeight - PaletteContainerStyle.GetVerticalFrameSize()
	m.QuickView.GlamourRenderer = m.GlamourRenderer
	m.QuickView.SetMessages([]types.Message{
		{Type: types.CommandResultMessage, Content: res.Payload},
	})
	m.QuickView.Viewport.SetContent(m.QuickView.renderContent())
	m.QuickView.Viewport.GotoTop()
	m.QuickView.needsRender = false
	m.ActiveOverlay = overlayQuickView
	m.Chat.TextArea.Blur()
	return m, nil
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
			if len(prefixParts) == 0 {
				prefixParts = determinePrefix(parts[0])
			}
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

func determinePrefix(part string) []string {
	if strings.HasPrefix(part, "/@") {
		return []string{"/@"}
	}
	if strings.HasPrefix(part, "/!") {
		return []string{"/!"}
	}
	if strings.HasPrefix(part, "@") {
		return []string{"@"}
	}
	if strings.HasPrefix(part, "!") {
		return []string{"!"}
	}
	return nil
}
