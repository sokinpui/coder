package ui

import (
	"fmt"
	"strings"
	"time"

	"coder/internal/core"
	"coder/internal/session"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	var (
		cmd tea.Cmd
	)

	if msg.Type != tea.KeyCtrlC {
		m.ctrlCPressed = false
	}

	// Handle scrolling regardless of state.
	switch msg.Type {
	case tea.KeyCtrlU:
		m.viewport.HalfViewUp()
		return m, nil, true
	case tea.KeyCtrlD:
		m.viewport.HalfViewDown()
		return m, nil, true
	}

	switch m.state {
	case stateThinking, stateGenerating, stateCancelling:
		switch msg.Type {
		case tea.KeyCtrlC:
			if m.state != stateCancelling {
				m.session.CancelGeneration()
				m.state = stateCancelling
			}
		case tea.KeyCtrlN:
			if m.isStreaming {
				m.session.CancelGeneration()
				m.isStreaming = false
				m.streamSub = nil
			}
			event := m.session.HandleInput(":new")
			if event.Type == session.NewSessionStarted {
				newModel, cmd := m.newSession()
				newModel.state = stateIdle
				return newModel, cmd, true
			}
			return m, nil, true
		case tea.KeyCtrlB:
			event := m.session.HandleInput(":branch")
			if event.Type == session.BranchModeStarted {
				m.visualIsSelecting = true
				m.state = stateVisualSelect
				m.visualMode = visualModeBranch
				m.selectableBlocks = groupMessages(m.session.GetMessages())
				if len(m.selectableBlocks) > 0 {
					m.visualSelectCursor = len(m.selectableBlocks) - 1
					m.visualSelectStart = m.visualSelectCursor
				}
				m.textArea.Reset()
				m.textArea.Blur()
				originalOffset := m.viewport.YOffset
				m.viewport.SetContent(m.renderConversation())
				m.viewport.SetYOffset(originalOffset)
			} else if event.Type == session.MessagesUpdated {
				// This handles the case where branching is not possible (e.g., no messages)
				// and an error message was added to the session.
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
			}
			return m, nil, true
		case tea.KeyCtrlH:
			m.state = stateHistorySelect
			m.textArea.Blur()
			return m, listHistoryCmd(m.session.GetHistoryManager()), true
		case tea.KeyEscape:
			// Allow entering visual mode even during generation.
			m.state = stateVisualSelect
			m.visualMode = visualModeNone
			m.visualIsSelecting = false
			m.selectableBlocks = groupMessages(m.session.GetMessages())
			if len(m.selectableBlocks) > 0 {
				m.visualSelectCursor = len(m.selectableBlocks) - 1
			}
			m.textArea.Blur()
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
		}
		return m, nil, true

	case stateHistorySelect:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.historyItems = nil
			if m.isStreaming {
				// Return to the generation view
				messages := m.session.GetMessages()
				if len(messages) > 0 && messages[len(messages)-1].Type == core.AIMessage && messages[len(messages)-1].Content == "" {
					m.state = stateThinking
				} else {
					m.state = stateGenerating
				}
				m.viewport.SetContent(m.renderConversation())
				// Re-issue commands needed for generation state
				return m, tea.Batch(listenForStream(m.streamSub), renderTick(), m.spinner.Tick), true
			} else {
				// Return to idle
				m.state = stateIdle
				m.textArea.Focus()
				m.viewport.SetContent(m.renderConversation())
				return m, textarea.Blink, true
			}

		case tea.KeyEnter:
			if len(m.historyItems) == 0 || m.historySelectCursor >= len(m.historyItems) {
				return m, nil, true
			}
			selectedItem := m.historyItems[m.historySelectCursor]
			if m.isStreaming {
				m.session.CancelGeneration()
				m.isStreaming = false // Prevent streamFinishedMsg from running
				m.streamSub = nil
			}
			return m, loadConversationCmd(m.session, selectedItem.Filename), true

		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "j":
				if m.historySelectCursor < len(m.historyItems)-1 {
					m.historySelectCursor++
					headerHeight := 10 // offset to bottom
					cursorLine := m.historySelectCursor + headerHeight
					if cursorLine >= m.viewport.YOffset+m.viewport.Height {
						m.viewport.LineDown(1)
					}
					m.viewport.SetContent(m.historyView()) // Re-render to update selection highlight
				}
				return m, nil, true
			case "k":
				if m.historySelectCursor > 0 {
					m.historySelectCursor--
					headerHeight := -10 // offset to top
					cursorLine := m.historySelectCursor + headerHeight
					if cursorLine < m.viewport.YOffset {
						m.viewport.LineUp(1)
					}
					m.viewport.SetContent(m.historyView()) // Re-render to update selection highlight
				}
				return m, nil, true
			}
		}
		return m, nil, true

	case stateVisualSelect:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			if m.visualIsSelecting {
				m.visualIsSelecting = false
				m.viewport.SetContent(m.renderConversation())
				return m, nil, true
			}
			var cmd tea.Cmd = textarea.Blink
			if m.isStreaming {
				messages := m.session.GetMessages()
				// Check if the last message is an empty AI message, which indicates 'thinking' state.
				if len(messages) > 0 && messages[len(messages)-1].Type == core.AIMessage && messages[len(messages)-1].Content == "" {
					m.state = stateThinking
				} else {
					m.state = stateGenerating
				}
				cmd = tea.Batch(cmd, m.spinner.Tick)
			} else {
				m.state = stateIdle
			}
			m.visualMode = visualModeNone
			m.textArea.Focus()
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, cmd, true

		case tea.KeyEnter:
			if m.visualSelectCursor >= len(m.selectableBlocks) {
				return m, nil, true // Out of bounds, do nothing
			}

			var cmds []tea.Cmd
			var cmd tea.Cmd

			switch m.visualMode {
			case visualModeGenerate:
				block := m.selectableBlocks[m.visualSelectCursor]
				userMsgIndex := -1
				// Find the first user message at or before the start of the selected block
				for i := block.startIdx; i >= 0; i-- {
					if m.session.GetMessages()[i].Type == core.UserMessage {
						userMsgIndex = i
						break
					}
				}

				if userMsgIndex != -1 {
					if m.isStreaming {
						m.session.CancelGeneration()
						m.isStreaming = false
						m.streamSub = nil
					}

					// Exit visual mode before starting generation
					m.state = stateIdle
					m.visualMode = visualModeNone
					m.textArea.Focus()

					event := m.session.RegenerateFrom(userMsgIndex)
					model, cmd := m.startGeneration(event)
					return model, cmd, true
				}
				// If no user message found (should be impossible if there are blocks),
				// fall through to exit visual mode.

			case visualModeEdit:
				block := m.selectableBlocks[m.visualSelectCursor]
				userMsgIndex := -1
				// Find the first user message at or before the start of the selected block
				for i := block.startIdx; i >= 0; i-- {
					if m.session.GetMessages()[i].Type == core.UserMessage {
						userMsgIndex = i
						break
					}
				}

				if userMsgIndex != -1 {
					// Exit visual mode before starting editor
					m.state = stateIdle
					m.visualMode = visualModeNone
					m.editingMessageIndex = userMsgIndex
					originalContent := m.session.GetMessages()[userMsgIndex].Content
					return m, editInEditorCmd(originalContent), true
				}
				// Fall through to exit visual mode if no user message found.
			case visualModeBranch:
				block := m.selectableBlocks[m.visualSelectCursor]
				endMessageIndex := block.endIdx

				if m.isStreaming {
					m.session.CancelGeneration()
					m.isStreaming = false
					m.streamSub = nil
				}

				newSess, err := m.session.Branch(endMessageIndex)
				if err != nil {
					m.statusBarMessage = fmt.Sprintf("Error branching session: %v", err)
					cmd = clearStatusBarCmd(5 * time.Second)
				} else {
					m.session = newSess
					m.statusBarMessage = "Branched to a new session."
					cmd = clearStatusBarCmd(2 * time.Second)

					// Exit visual mode and apply changes
					m.state = stateIdle
					m.visualMode = visualModeNone
					m.visualIsSelecting = false

					// Reset UI state for new session
					m.lastInteractionFailed = false
					m.lastRenderedAIPart = ""
					m.textArea.Reset()
					m.textArea.SetHeight(1)
					m.textArea.Focus()
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()

					// Recalculate token count
					m.isCountingTokens = true
					cmds = append(cmds, countTokensCmd(m.session.GetPromptForTokenCount()))

					return m, tea.Batch(textarea.Blink, cmd, tea.Batch(cmds...)), true
				}
				// On error, fall through to exit visual mode.
			default:
				// For visualModeNone, Enter does nothing.
				return m, nil, true
			}

			m.state = stateIdle
			m.visualMode = visualModeNone
			m.visualIsSelecting = false
			m.textArea.Focus()
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, tea.Batch(textarea.Blink, cmd, tea.Batch(cmds...)), true

		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "j":
				if m.visualSelectCursor < len(m.selectableBlocks)-1 {
					m.visualSelectCursor++
					offset := m.viewport.YOffset
					m.viewport.SetContent(m.renderConversation())
					m.viewport.SetYOffset(offset)
				}
				// Allow the viewport to handle the key press for scrolling
				return m, nil, false
			case "k":
				if m.visualSelectCursor > 0 {
					m.visualSelectCursor--
					offset := m.viewport.YOffset
					m.viewport.SetContent(m.renderConversation())
					m.viewport.SetYOffset(offset)
				}
				// Allow the viewport to handle the key press for scrolling
				return m, nil, false
			case "v", "V":
				if !m.visualIsSelecting {
					m.visualIsSelecting = true
					m.visualSelectStart = m.visualSelectCursor
					m.viewport.SetContent(m.renderConversation())
				}
			case "b":
				if !m.visualIsSelecting && m.visualMode == visualModeNone {
					m.visualMode = visualModeBranch
					m.visualIsSelecting = true // branch is a single-selection mode
					m.viewport.SetContent(m.renderConversation())
					return m, nil, true
				}
			case "n":
				if m.isStreaming {
					m.session.CancelGeneration()
					m.isStreaming = false
					m.streamSub = nil
				}
				event := m.session.HandleInput(":new")
				if event.Type == session.NewSessionStarted {
					newModel, cmd := m.newSession()
					newModel.state = stateIdle
					newModel.visualMode = visualModeNone
					newModel.visualIsSelecting = false
					return newModel, cmd, true
				}
				return m, nil, true
			case "i":
				m.state = stateIdle
				m.visualMode = visualModeNone
				m.visualIsSelecting = false
				m.textArea.Focus()
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				return m, textarea.Blink, true
			case "g":
				if !m.visualIsSelecting && m.visualMode == visualModeNone {
					if m.visualSelectCursor >= len(m.selectableBlocks) {
						return m, nil, true // Out of bounds
					}
					block := m.selectableBlocks[m.visualSelectCursor]
					userMsgIndex := -1
					// Find the first user message at or before the start of the selected block
					for i := block.startIdx; i >= 0; i-- {
						if m.session.GetMessages()[i].Type == core.UserMessage {
							userMsgIndex = i
							break
						}
					}

					if userMsgIndex != -1 {
						if m.isStreaming {
							m.session.CancelGeneration()
							m.isStreaming = false
							m.streamSub = nil
						}

						// Exit visual mode before starting generation
						m.state = stateIdle
						m.visualMode = visualModeNone
						m.textArea.Focus()

						event := m.session.RegenerateFrom(userMsgIndex)
						model, cmd := m.startGeneration(event)
						return model, cmd, true
					}
				}
			case "e":
				if !m.visualIsSelecting && m.visualMode == visualModeNone {
					if m.visualSelectCursor >= len(m.selectableBlocks) {
						return m, nil, true // Out of bounds
					}
					block := m.selectableBlocks[m.visualSelectCursor]
					userMsgIndex := -1
					// Find the first user message at or before the start of the selected block
					for i := block.startIdx; i >= 0; i-- {
						if m.session.GetMessages()[i].Type == core.UserMessage {
							userMsgIndex = i
							break
						}
					}

					if userMsgIndex != -1 {
						m.state = stateIdle
						m.visualMode = visualModeNone
						m.editingMessageIndex = userMsgIndex
						originalContent := m.session.GetMessages()[userMsgIndex].Content
						return m, editInEditorCmd(originalContent), true
					}
				}
			case "y":
				if m.visualIsSelecting && m.visualMode == visualModeNone {
					start, end := m.visualSelectStart, m.visualSelectCursor
					if start > end {
						start, end = end, start
					}
					var selectedMessages []core.Message
					for i := start; i <= end; i++ {
						block := m.selectableBlocks[i]
						for j := block.startIdx; j <= block.endIdx; j++ {
							selectedMessages = append(selectedMessages, m.session.GetMessages()[j])
						}
					}
					content := core.BuildHistorySnippet(selectedMessages)
					if err := clipboard.WriteAll(content); err == nil {
						m.statusBarMessage = "Copied to clipboard."
						cmd = clearStatusBarCmd(2 * time.Second)
					}
					m.state = stateIdle
					m.visualMode = visualModeNone
					m.visualIsSelecting = false
					m.textArea.Focus()
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					return m, tea.Batch(textarea.Blink, cmd), true
				}
			case "d":
				if m.visualIsSelecting && m.visualMode == visualModeNone {
					start, end := m.visualSelectStart, m.visualSelectCursor
					if start > end {
						start, end = end, start
					}
					var selectedIndices []int
					for i := start; i <= end; i++ {
						block := m.selectableBlocks[i]
						for j := block.startIdx; j <= block.endIdx; j++ {
							selectedIndices = append(selectedIndices, j)
						}
					}

					isDeletingCurrentAIMessage := false
					if m.isStreaming && len(m.session.GetMessages()) > 0 {
						lastMessageIndex := len(m.session.GetMessages()) - 1
						for _, idx := range selectedIndices {
							if idx == lastMessageIndex {
								isDeletingCurrentAIMessage = true
								break
							}
						}
					}
					if isDeletingCurrentAIMessage {
						m.session.CancelGeneration()
						m.isStreaming = false
						m.streamSub = nil
					}
					m.session.DeleteMessages(selectedIndices)
					m.statusBarMessage = "Deleted selected messages."
					cmd = clearStatusBarCmd(2 * time.Second)
					m.state = stateIdle
					m.visualMode = visualModeNone
					m.visualIsSelecting = false
					m.textArea.Focus()
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					return m, tea.Batch(textarea.Blink, cmd), true
				}
			}
		}
		return m, nil, true

	case stateIdle:
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
				// Enter visual mode.
				m.state = stateVisualSelect
				m.visualMode = visualModeNone
				m.visualIsSelecting = false
				m.selectableBlocks = groupMessages(m.session.GetMessages())
				if len(m.selectableBlocks) > 0 {
					m.visualSelectCursor = len(m.selectableBlocks) - 1
				}
				m.textArea.Blur()
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				return m, nil, true
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
				m.visualIsSelecting = true
				m.state = stateVisualSelect
				m.visualMode = visualModeBranch
				m.selectableBlocks = groupMessages(m.session.GetMessages())
				if len(m.selectableBlocks) > 0 {
					m.visualSelectCursor = len(m.selectableBlocks) - 1
					m.visualSelectStart = m.visualSelectCursor
				}
				m.textArea.Reset()
				m.textArea.Blur()
				originalOffset := m.viewport.YOffset
				m.viewport.SetContent(m.renderConversation())
				m.viewport.SetYOffset(originalOffset)
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
	}
	return m, nil, false
}
