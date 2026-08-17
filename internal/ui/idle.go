package ui

import (
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

func (m Model) handleEvent(event types.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case types.NoOp:
		return m, nil

	case types.MessagesUpdated:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m = m.updateLayout()
		m.UpdateTokenCount()
		return m, nil

	case types.NewSessionStarted:
		return m.newSession(event.Mode)

	case types.GenerationStarted:
		return m.startGeneration(event)

	case types.AtomicMsgModeStarted,
		types.GenerateModeStarted,
		types.EditModeStarted,
		types.BranchModeStarted:
		return m.openAtomicMsgMode()
	case types.FzfModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.ActiveOverlay = overlayFinder
		m.Chat.TextArea.Blur()
		var items []string
		items = append(items, m.Session.GetConfig().AvailableModels...)
		m.Finder.AllItems = items
		m.Finder.FoundItems = items
		m.Finder.Cursor = 0
		if payload, ok := event.Data.(string); ok && payload != "" {
			m.Finder.TextInput.SetValue(payload)
		} else {
			m.Finder.TextInput.Reset()
		}
		m.Finder.updateFoundItems()
		m.Finder.TextInput.Focus()
		m.UpdateTokenCount()
		return m, textinput.Blink
	case types.HistoryModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.ActiveOverlay = overlayHistory
		m.History.Tab = TabHistory
		m.History.SearchInput.Reset()
		m.History.IsSearching = false
		m.Chat.TextArea.Blur()
		m.UpdateTokenCount()
		return m, tea.Batch(listHistoryCmd(m.Session.GetHistoryManager()), m.Chat.Spinner.Tick)
	case types.ActiveModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.ActiveOverlay = overlayHistory
		m.History.Tab = TabActive
		m.History.SearchInput.Reset()
		m.History.IsSearching = false
		m.Chat.TextArea.Blur()
		m.UpdateTokenCount()
		m.updateActiveFilter()
		return m, tea.Batch(listHistoryCmd(m.Session.GetHistoryManager()), m.Chat.Spinner.Tick)
	case types.HelpViewerStarted, types.ConfigViewerStarted, types.ModelViewerStarted, types.ListViewerStarted:
		cmdName := "/help"
		switch event.Type {
		case types.ConfigViewerStarted:
			cmdName = "/config"
		case types.ModelViewerStarted:
			cmdName = "/model"
		case types.ListViewerStarted:
			cmdName = "/list"
		}
		m.QuickView.SetMessages([]types.Message{
			{Type: types.CommandMessage, Content: cmdName},
			{Type: types.CommandResultMessage, Content: event.Data.(string)},
		})
		m.ActiveOverlay = overlayQuickView
		m.Chat.TextArea.Blur()
		return m, nil
	case types.ExternalEditorStarted:
		return m, openInEditorCmd(event.Data.(string))

	case types.Quit:
		m.Quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) newSession(mode string) (Model, tea.Cmd) {
	if err := m.Session.SaveConversation(); err != nil {
		log.Printf("Error saving conversation before switching: %v", err)
	}

	if mode == "" {
		mode = "coding"
	}

	newSess, err := session.New(m.Session.GetConfig(), mode, m.Session.GetInstruction(), m.Session.GetContextFiles())
	if err != nil {
		log.Printf("Error creating new session: %v", err)
		return m, nil
	}

	m.Session = newSess
	m.ClearCache()
	m.addActiveSession(newSess)
	m.Session.AddMessages(types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage})
	dirMsg := utils.GetDirInfoContent()
	m.Session.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})

	m.Chat.LastInteractionFailed = false
	m.Chat.TextArea.Focus()
	m.Chat.Viewport.GotoTop()
	m.Chat.Viewport.SetContent(m.renderConversation())

	m.UpdateTokenCount()
	return m, loadInitialContextCmd(m.Session)
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.Chat.TextArea.Value()

	// don't send if the input is empty
	if strings.TrimSpace(input) == "" {
		return m, nil
	}

	if !strings.HasPrefix(input, "/") {
		m.Session.AddMessages(types.Message{Type: types.UserMessage, Content: input})
		m.UpdateTokenCount()
		m.Chat.ShowPalette = false

		var cmds []tea.Cmd
		if !m.Session.IsTitleGenerated() {
			cmds = append(cmds, generateTitleCmd(m.Session, input))
		}

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
		event := m.Session.HandleShortcut("/history")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

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
		event := m.Session.HandleShortcut("/branch")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case km.Finder:
		event := m.Session.HandleShortcut("/fzf")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

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
