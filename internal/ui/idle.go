package ui

import (
	"github.com/sokinpui/coder/internal/types"
	"fmt"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleEvent(event types.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case types.NoOp:
		return m, nil

	case types.MessagesUpdated:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m = m.updateLayout()
		m.IsCountingTokens = true
		return m, countTokensCmd(m.Session.GetPrompt())

	case types.NewSessionStarted:
		return m.newSession()

	case types.GenerationStarted:
		return m.startGeneration(event)

	case types.VisualModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m.enterVisualMode(visualModeNone)

	case types.GenerateModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m.enterVisualMode(visualModeGenerate)
	case types.EditModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m.enterVisualMode(visualModeEdit)
	case types.BranchModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m.enterVisualMode(visualModeBranch)
	case types.SearchModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.State = stateSearch
		m.Chat.TextArea.Blur()
		m.Search.AllItems = m.collectSearchableMessages()
		m.Search.TextInput.SetValue(event.Data.(string))
		m.Search.updateFoundItems()
		m.Search.Visible = true
		m.Search.TextInput.Focus()
		m.IsCountingTokens = true
		return m, tea.Batch(textinput.Blink, countTokensCmd(m.Session.GetPrompt()))
	case types.FzfModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.State = stateFinder
		m.Chat.TextArea.Blur()
		var items []string
		for _, model := range m.Session.GetConfig().AvailableModels {
			items = append(items, fmt.Sprintf("model: %s", model))
		}
		m.Finder.AllItems = items
		m.Finder.FoundItems = items
		m.Finder.Visible = true
		m.Finder.TextInput.Focus()
		m.IsCountingTokens = true
		return m, tea.Batch(textinput.Blink, countTokensCmd(m.Session.GetPrompt()))
	case types.HistoryModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.State = stateHistorySelect
		m.Chat.TextArea.Blur()
		m.IsCountingTokens = true
		return m, tea.Batch(listHistoryCmd(m.Session.GetHistoryManager()), countTokensCmd(m.Session.GetPrompt()), m.Chat.Spinner.Tick)
	case types.TreeModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.State = stateTree
		m.Chat.TextArea.Blur()
		m.Tree.Visible = true
		m.Tree.loadInitialSelection(m.Session.GetConfig())
		m.IsCountingTokens = true
		return m, tea.Batch(m.Tree.initCmd(), countTokensCmd(m.Session.GetPrompt()))
	case types.JumpModeStarted:
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.State = stateJump
		m.Chat.TextArea.Blur()
		m.Jump.Visible = true
		m.Jump.UpdateItems(m.Session.GetMessages())
		m.IsCountingTokens = true
		return m, tea.Batch(countTokensCmd(m.Session.GetPrompt()))

	case types.Quit:
		m.Quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) newSession() (Model, tea.Cmd) {
	// The session handles saving and clearing messages.
	// The UI just needs to reset its state.
	m.Session.AddMessages(types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage})
	dirMsg := utils.GetDirInfoContent()
	m.Session.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})

	// Reset UI and state flags.
	m.Chat.LastInteractionFailed = false
	m.Chat.LastRenderedAIPart = ""
	m.Chat.TextArea.Focus()
	m.Chat.Viewport.GotoTop()
	m.Chat.Viewport.SetContent(m.renderConversation())

	// Recalculate the token count for the base context.
	m.IsCountingTokens = true
	return m, countTokensCmd(m.Session.GetPrompt())
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.Chat.TextArea.Value()

	// don't send if the input is empty
	if strings.TrimSpace(input) == "" {
		return m, nil
	}

	// Clear search focus and query when submitting a new interaction
	m.Chat.SearchQuery = ""
	m.Chat.SearchFocusMsgIndex = -1
	m.Chat.SearchFocusLineNum = -1

	if !strings.HasPrefix(input, ":") {
		// This is a prompt, apply debounce.
		m.Session.AddMessages(types.Message{Type: types.UserMessage, Content: input})

		var cmds []tea.Cmd
		if !m.Session.IsTitleGenerated() {
			cmds = append(cmds, generateTitleCmd(m.Session, input))
		}

		m.State = stateGenPending
		m.Chat.TextArea.Blur()
		m.Chat.TextArea.Reset()
		m = m.updateLayout()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()

		cmds = append(cmds, tea.Tick(1*time.Second, func(t time.Time) tea.Msg { return startGenerationMsg{} }), m.Chat.Spinner.Tick)
		return m, tea.Batch(cmds...)
	}

	// This is a command, handle as before.
	if len(m.Chat.CommandHistory) == 0 || m.Chat.CommandHistory[len(m.Chat.CommandHistory)-1] != input {
		m.Chat.CommandHistory = append(m.Chat.CommandHistory, input)
	}
	m.Chat.CommandHistoryCursor = len(m.Chat.CommandHistory)
	m.Chat.CommandHistoryModified = ""
	event := m.Session.HandleInput(input)

	shouldPreserve := m.Chat.PreserveInputOnSubmit
	m.Chat.PreserveInputOnSubmit = false

	model, cmd := m.handleEvent(event)
	if newModel, ok := model.(Model); ok {
		isCommand := strings.HasPrefix(input, ":")
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
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		if strings.HasPrefix(m.Chat.TextArea.Value(), ":") {
			if m.Chat.CommandHistoryCursor == len(m.Chat.CommandHistory) {
				m.Chat.CommandHistoryModified = m.Chat.TextArea.Value()
			}

			if msg.Type == tea.KeyUp {
				if m.Chat.CommandHistoryCursor > 0 {
					m.Chat.CommandHistoryCursor--
					m.Chat.TextArea.SetValue(m.Chat.CommandHistory[m.Chat.CommandHistoryCursor])
					m = m.updateLayout()
					m.Chat.TextArea.CursorEnd()
				}
			} else { // KeyDown
				if m.Chat.CommandHistoryCursor < len(m.Chat.CommandHistory) {
					m.Chat.CommandHistoryCursor++
					if m.Chat.CommandHistoryCursor == len(m.Chat.CommandHistory) {
						m.Chat.TextArea.SetValue(m.Chat.CommandHistoryModified)
						m = m.updateLayout()
					} else {
						m.Chat.TextArea.SetValue(m.Chat.CommandHistory[m.Chat.CommandHistoryCursor])
						m = m.updateLayout()
					}
					m.Chat.TextArea.CursorEnd()
				}
			}
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

	case tea.KeyEscape:
		// Enter visual mode, preserving the text in the text area.
		model, cmd := m.enterVisualMode(visualModeNone)
		return model, cmd, true

	case tea.KeyTab, tea.KeyShiftTab:
		isCommand := strings.HasPrefix(m.Chat.TextArea.Value(), ":")
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
				m.Chat.TextArea.SetValue(strings.Join(append(prefixParts, selectedItem), " "))
				m = m.updateLayout()
			} else {
				m.Chat.TextArea.SetValue(selectedItem)
				m = m.updateLayout()
			}
			m.Chat.TextArea.CursorEnd()
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
				m.Chat.TextArea.SetValue(strings.Join(append(prefixParts, selectedItem), " "))
			} else {
				m.Chat.TextArea.SetValue(selectedItem)
			}
			m.Chat.TextArea.CursorEnd()
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}

		// Smart enter: submit if it's a command.
		if strings.HasPrefix(m.Chat.TextArea.Value(), ":") {
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}
		// Otherwise, fall through to let the textarea handle the newline.
		return m, nil, false

	case tea.KeyCtrlH:
		event := m.Session.HandleInput(":history")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case tea.KeyCtrlE:
		if m.Chat.TextArea.Focused() {
			return m, editInEditorCmd(m.Chat.TextArea.Value()), true
		}

	case tea.KeyCtrlJ:
		model, cmd := m.handleSubmit()
		return model, cmd, true

	case tea.KeyCtrlN:
		event := m.Session.HandleInput(":new")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case tea.KeyCtrlB:
		event := m.Session.HandleInput(":branch")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case tea.KeyCtrlF:
		event := m.Session.HandleInput(":fzf")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case tea.KeyCtrlT:
		event := m.Session.HandleInput(":tree")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case tea.KeyCtrlA:
		// Equivalent to typing ":itf" and pressing enter.
		event := m.Session.HandleInput(":itf")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case tea.KeyCtrlP:
		event := m.Session.HandleInput(":search")
		model, cmd := m.handleEvent(event)
		return model, cmd, true

	case tea.KeyCtrlV:
		return m, handlePasteCmd(m.Session.GetConfig()), true
	}
	return m, nil, false
}
