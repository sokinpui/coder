package ui

import (
	"fmt"
	"slices"

	"coder/internal/commands"
	"coder/internal/history"
	"coder/internal/types"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) enterVisualMode(mode visualMode) (Model, tea.Cmd) {
	m.State = stateVisualSelect
	m.VisualMode = mode
	m.SelectableBlocks = groupMessages(m.Session.GetMessages())

	isSelectionMode := mode != visualModeNone
	m.VisualIsSelecting = isSelectionMode

	if len(m.SelectableBlocks) > 0 {
		m.VisualSelectCursor = len(m.SelectableBlocks) - 1
		if isSelectionMode {
			m.VisualSelectStart = m.VisualSelectCursor
		}
	}

	if isSelectionMode {
		m.TextArea.Reset()
	}
	m.TextArea.Blur()

	originalOffset := m.Viewport.YOffset
	m.Viewport.SetContent(m.renderConversation())

	if isSelectionMode {
		m.Viewport.SetYOffset(originalOffset)
	} else {
		m.Viewport.GotoBottom()
	}

	return m, nil
}

func (m Model) handleKeyPressVisual(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg.Type {
	case tea.KeyCtrlA:
		if m.VisualSelectCursor >= len(m.SelectableBlocks) {
			return m, nil, true // Out of bounds
		}

		cursorBlock := m.SelectableBlocks[m.VisualSelectCursor]
		var aiResponseToApply string
		found := false
		messages := m.Session.GetMessages()

		// Search backwards from the end of current cursor's block
		for i := cursorBlock.endIdx; i >= 0; i-- {
			if messages[i].Type == types.AIMessage && messages[i].Content != "" {
				aiResponseToApply = messages[i].Content
				found = true
				break
			}
		}

		if !found {
			m.StatusBarMessage = "No AI response found above the cursor to apply."
			return m, clearStatusBarCmd(), true
		}

		// Execute itf
		result, success := commands.ExecuteItf(aiResponseToApply, "")

		// Add messages to show command execution
		m.Session.AddMessages(types.Message{
			Type:    types.CommandMessage,
			Content: ":itf (from visual mode)",
		})
		if success {
			m.Session.AddMessages(types.Message{Type: types.CommandResultMessage, Content: result})
		} else {
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: result})
		}

		// Exit visual mode and update UI
		m.State = stateIdle
		m.VisualMode = visualModeNone
		m.TextArea.Focus()
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()
		return m, textarea.Blink, true
	case tea.KeyEsc, tea.KeyCtrlC:
		if m.VisualIsSelecting {
			m.VisualIsSelecting = false
			m.Viewport.SetContent(m.renderConversation())
			return m, nil, true
		}
		var cmd tea.Cmd = textarea.Blink
		if m.IsStreaming {
			messages := m.Session.GetMessages()
			// Check if the last message is an empty AI message, which indicates 'thinking' state.
			if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
				m.State = stateThinking
			} else {
				m.State = stateGenerating
			}
			cmd = tea.Batch(cmd, m.Spinner.Tick)
		} else {
			m.State = stateIdle
		}
		m.VisualMode = visualModeNone
		m.TextArea.Focus()
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()
		return m, cmd, true

	case tea.KeyEnter:
		if m.VisualSelectCursor >= len(m.SelectableBlocks) {
			return m, nil, true // Out of bounds, do nothing
		}

		switch m.VisualMode {
		case visualModeGenerate:
			block := m.SelectableBlocks[m.VisualSelectCursor]
			msgIndex := -1
			// Find the first user or image message at or before the start of the selected block
			for i := block.startIdx; i >= 0; i-- {
				msgType := m.Session.GetMessages()[i].Type
				if msgType == types.UserMessage || msgType == types.ImageMessage {
					msgIndex = i
					break
				}
			}

			if msgIndex != -1 {
				if m.IsStreaming {
					m.Session.CancelGeneration()
					m.IsStreaming = false
					m.StreamSub = nil
				}

				// Exit visual mode before starting generation
				m.State = stateIdle
				m.VisualMode = visualModeNone
				m.TextArea.Focus()

				event := m.Session.RegenerateFrom(msgIndex)
				model, cmd := m.startGeneration(event)
				return model, cmd, true
			}
			// If no user message found (should be impossible if there are blocks),
			// fall through to exit visual mode.

		case visualModeEdit:
			block := m.SelectableBlocks[m.VisualSelectCursor]
			userMsgIndex := -1
			// Find the first user message at or before the start of the selected block
			for i := block.startIdx; i >= 0; i-- {
				if m.Session.GetMessages()[i].Type == types.UserMessage {
					userMsgIndex = i
					break
				}
			}

			if userMsgIndex != -1 {
				// Exit visual mode before starting editor
				m.State = stateIdle
				m.VisualMode = visualModeNone
				m.EditingMessageIndex = userMsgIndex
				originalContent := m.Session.GetMessages()[userMsgIndex].Content
				return m, editInEditorCmd(originalContent), true
			}
			// Fall through to exit visual mode if no user message found.
		case visualModeBranch:
			block := m.SelectableBlocks[m.VisualSelectCursor]
			endMessageIndex := block.endIdx

			if m.IsStreaming {
				m.Session.CancelGeneration()
				m.IsStreaming = false
				m.StreamSub = nil
			}

			newSess, err := m.Session.Branch(endMessageIndex)
			if err != nil {
				m.StatusBarMessage = fmt.Sprintf("Error branching session: %v", err)
				cmd = clearStatusBarCmd()
			} else {
				m.Session = newSess
				m.StatusBarMessage = "Branched to a new session."
				cmd = clearStatusBarCmd()

				// Exit visual mode and apply changes
				m.State = stateIdle
				m.VisualMode = visualModeNone
				m.VisualIsSelecting = false

				// Reset UI state for new session
				m.LastInteractionFailed = false
				m.LastRenderedAIPart = ""
				m.TextArea.Reset()
				m.TextArea.SetHeight(1)
				m.TextArea.Focus()
				m.Viewport.SetContent(m.renderConversation())
				m.Viewport.GotoBottom()

				// Recalculate token count
				m.IsCountingTokens = true
				cmds = append(cmds, countTokensCmd(m.Session.GetPrompt()))

				return m, tea.Batch(textarea.Blink, cmd, tea.Batch(cmds...)), true
			}
			// On error, fall through to exit visual mode.
		default:
			// For visualModeNone, Enter does nothing.
			return m, nil, true
		}

		m.State = stateIdle
		m.VisualMode = visualModeNone
		m.VisualIsSelecting = false
		m.TextArea.Focus()
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()
		return m, tea.Batch(textarea.Blink, cmd, tea.Batch(cmds...)), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			if m.VisualSelectCursor < len(m.SelectableBlocks)-1 {
				m.VisualSelectCursor++
				offset := m.Viewport.YOffset
				m.Viewport.SetContent(m.renderConversation())
				m.Viewport.SetYOffset(offset)
			}
			// Allow the viewport to handle the key press for scrolling
			return m, nil, false
		case "k":
			if m.VisualSelectCursor > 0 {
				m.VisualSelectCursor--
				offset := m.Viewport.YOffset
				m.Viewport.SetContent(m.renderConversation())
				m.Viewport.SetYOffset(offset)
			}
			// Allow the viewport to handle the key press for scrolling
			return m, nil, false
		case "o", "O":
			if m.VisualIsSelecting {
				m.VisualSelectCursor, m.VisualSelectStart = m.VisualSelectStart, m.VisualSelectCursor
				m.Viewport.SetContent(m.renderConversation())
			}
			return m, nil, true
		case "v", "V":
			if m.VisualIsSelecting {
				m.VisualIsSelecting = false
			} else {
				m.VisualIsSelecting = true
				m.VisualSelectStart = m.VisualSelectCursor
			}
			m.Viewport.SetContent(m.renderConversation())
			return m, nil, true
		case "b":
			if !m.VisualIsSelecting && m.VisualMode == visualModeNone {
				m.VisualMode = visualModeBranch
				m.VisualIsSelecting = true // branch is a single-selection mode
				m.Viewport.SetContent(m.renderConversation())
				return m, nil, true
			}
		case "n":
			if m.IsStreaming {
				m.Session.CancelGeneration()
				m.IsStreaming = false
				m.StreamSub = nil
			}
			event := m.Session.HandleInput(":new")
			if event.Type == types.NewSessionStarted {
				newModel, cmd := m.newSession()
				newModel.State = stateIdle
				newModel.VisualMode = visualModeNone
				newModel.VisualIsSelecting = false
				return newModel, cmd, true
			}
			return m, nil, true
		case "i":
			m.State = stateIdle
			m.VisualMode = visualModeNone
			m.VisualIsSelecting = false
			m.TextArea.Focus()
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
			return m, textarea.Blink, true
		case "g":
			if !m.VisualIsSelecting && m.VisualMode == visualModeNone {
				if m.VisualSelectCursor >= len(m.SelectableBlocks) {
					return m, nil, true // Out of bounds
				}
				block := m.SelectableBlocks[m.VisualSelectCursor]
				msgIndex := -1
				// Find the first user or image message at or before the start of the selected block
				for i := block.startIdx; i >= 0; i-- {
					msgType := m.Session.GetMessages()[i].Type
					if msgType == types.UserMessage || msgType == types.ImageMessage {
						msgIndex = i
						break
					}
				}

				if msgIndex != -1 {
					if m.IsStreaming {
						m.Session.CancelGeneration()
						m.IsStreaming = false
						m.StreamSub = nil
					}

					// Exit visual mode before starting generation
					m.State = stateIdle
					m.VisualMode = visualModeNone
					m.TextArea.Focus()

					event := m.Session.RegenerateFrom(msgIndex)
					model, cmd := m.startGeneration(event)
					return model, cmd, true
				}
			}
		case "e":
			if !m.VisualIsSelecting && m.VisualMode == visualModeNone {
				if m.VisualSelectCursor >= len(m.SelectableBlocks) {
					return m, nil, true // Out of bounds
				}
				block := m.SelectableBlocks[m.VisualSelectCursor]
				userMsgIndex := -1
				// Find the first user message at or before the start of the selected block
				for i := block.startIdx; i >= 0; i-- {
					if m.Session.GetMessages()[i].Type == types.UserMessage {
						userMsgIndex = i
						break
					}
				}

				if userMsgIndex != -1 {
					m.EditingMessageIndex = userMsgIndex
					originalContent := m.Session.GetMessages()[userMsgIndex].Content
					return m, editInEditorCmd(originalContent), true
				}
			}
		case "y":
			if m.VisualIsSelecting && m.VisualMode == visualModeNone {
				start, end := m.VisualSelectStart, m.VisualSelectCursor
				if start > end {
					start, end = end, start
				}
				var selectedMessages []types.Message
				for i := start; i <= end; i++ {
					block := m.SelectableBlocks[i]
					for j := block.startIdx; j <= block.endIdx; j++ {
						selectedMessages = append(selectedMessages, m.Session.GetMessages()[j])
					}
				}

				var content string
				if len(selectedMessages) == 1 && selectedMessages[0].Type == types.UserMessage {
					content = selectedMessages[0].Content
				} else {
					content = history.BuildHistorySnippet(selectedMessages)
				}
				if err := clipboard.WriteAll(content); err == nil {
					m.StatusBarMessage = "Copied to clipboard."
					cmd = clearStatusBarCmd()
				}
				if m.IsStreaming {
					messages := m.Session.GetMessages()
					if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
						m.State = stateThinking
					} else {
						m.State = stateGenerating
					}
					cmd = tea.Batch(cmd, m.Spinner.Tick)
				} else {
					m.State = stateIdle
				}
				m.VisualMode = visualModeNone
				m.VisualIsSelecting = false
				m.TextArea.Focus()
				m.Viewport.SetContent(m.renderConversation())
				m.Viewport.GotoBottom()
				return m, tea.Batch(textarea.Blink, cmd), true
			}
		case "d":
			if m.VisualIsSelecting && m.VisualMode == visualModeNone {
				start, end := m.VisualSelectStart, m.VisualSelectCursor
				if start > end {
					start, end = end, start
				}
				var selectedIndices []int
				for i := start; i <= end; i++ {
					block := m.SelectableBlocks[i]
					for j := block.startIdx; j <= block.endIdx; j++ {
						selectedIndices = append(selectedIndices, j)
					}
				}

				isDeletingCurrentAIMessage := false
				if m.IsStreaming && len(m.Session.GetMessages()) > 0 {
					lastMessageIndex := len(m.Session.GetMessages()) - 1

					if slices.Contains(selectedIndices, lastMessageIndex) {
						isDeletingCurrentAIMessage = true
					}
				}
				if isDeletingCurrentAIMessage {
					m.Session.CancelGeneration()
					m.IsStreaming = false
					m.StreamSub = nil
				}
				m.Session.DeleteMessages(selectedIndices)
				m.StatusBarMessage = "Deleted selected messages."
				cmd = clearStatusBarCmd()
				if m.IsStreaming {
					messages := m.Session.GetMessages()
					if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
						m.State = stateThinking
					} else {
						m.State = stateGenerating
					}
					cmd = tea.Batch(cmd, m.Spinner.Tick)
				} else {
					m.State = stateIdle
				}
				m.VisualMode = visualModeNone
				m.VisualIsSelecting = false
				m.TextArea.Focus()
				m.Viewport.SetContent(m.renderConversation())
				m.Viewport.GotoBottom()
				m.IsCountingTokens = true
				return m, tea.Batch(textarea.Blink, cmd, countTokensCmd(m.Session.GetPrompt())), true
			}
		}
	}
	return m, nil, true
}
