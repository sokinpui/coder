package ui

import (
	"fmt"
	"time"

	"coder/internal/core"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) enterVisualMode(mode visualMode) (Model, tea.Cmd) {
	m.state = stateVisualSelect
	m.visualMode = mode
	m.selectableBlocks = groupMessages(m.session.GetMessages())

	isSelectionMode := mode != visualModeNone
	m.visualIsSelecting = isSelectionMode

	if len(m.selectableBlocks) > 0 {
		m.visualSelectCursor = len(m.selectableBlocks) - 1
		if isSelectionMode {
			m.visualSelectStart = m.visualSelectCursor
		}
	}

	if isSelectionMode {
		m.textArea.Reset()
	}
	m.textArea.Blur()

	originalOffset := m.viewport.YOffset
	m.viewport.SetContent(m.renderConversation())

	if isSelectionMode {
		m.viewport.SetYOffset(originalOffset)
	} else {
		m.viewport.GotoBottom()
	}

	return m, nil
}

func (m Model) handleKeyPressVisual(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
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
			if m.visualIsSelecting {
				m.visualIsSelecting = false
			} else {
				m.visualIsSelecting = true
				m.visualSelectStart = m.visualSelectCursor
			}
			m.viewport.SetContent(m.renderConversation())
			return m, nil, true
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
			if event.Type == core.NewSessionStarted {
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
				if m.isStreaming {
					messages := m.session.GetMessages()
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
				if m.isStreaming {
					messages := m.session.GetMessages()
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
				m.visualIsSelecting = false
				m.textArea.Focus()
				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				return m, tea.Batch(textarea.Blink, cmd), true
			}
		}
	}
	return m, nil, true
}
