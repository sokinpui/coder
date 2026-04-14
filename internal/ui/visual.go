package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type visualMode int

const (
	visualModeNone visualMode = iota
	visualModeGenerate
	visualModeEdit
	visualModeBranch
)

type messageBlock struct {
	startIdx int
	endIdx   int
}

type VisualSelectModel struct {
	Mode        visualMode
	IsSelecting bool
	Blocks      []messageBlock
	Cursor      int
	Start       int
}

func NewVisualSelect() VisualSelectModel {
	return VisualSelectModel{
		Mode: visualModeNone,
	}
}

func (m Model) enterVisualMode(mode visualMode) (Model, tea.Cmd) {
	blocks := groupMessages(m.Session.GetMessages())
	if len(blocks) == 0 {
		return m, nil
	}

	m.State = stateVisualSelect
	m.VisualSelect.Mode = mode
	m.VisualSelect.Blocks = blocks

	isSelectionMode := mode != visualModeNone
	m.VisualSelect.IsSelecting = isSelectionMode

	if len(m.VisualSelect.Blocks) > 0 {
		m.VisualSelect.Cursor = len(m.VisualSelect.Blocks) - 1
		if isSelectionMode {
			m.VisualSelect.Start = m.VisualSelect.Cursor
		}
	}

	if isSelectionMode {
		m.Chat.TextArea.Reset()
	}
	m.Chat.TextArea.Blur()

	originalOffset := m.Chat.Viewport.YOffset
	m.Chat.Viewport.SetContent(m.renderConversation())

	if isSelectionMode {
		m.Chat.Viewport.SetYOffset(originalOffset)
	} else {
		m.Chat.Viewport.GotoBottom()
	}

	return m, nil
}

func (m Model) handleKeyPressVisual(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	keyStr := msg.String()
	km := m.Session.GetConfig().Keymap

	switch keyStr {
	case km.ApplyITF:
		cursorBlock := m.getCurrentBlock()
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

		if success {
			m.Session.AddMessages(types.Message{Type: types.CommandResultMessage, Content: result})
		} else {
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: result})
		}

		// Exit visual mode and update UI
		m.State = stateIdle
		m.VisualSelect.Mode = visualModeNone
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m, textarea.Blink, true
	case km.History:
		event := m.Session.HandleShortcut(":history")
		model, cmd := m.handleEvent(event)
		return model, cmd, true
	}

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		if m.VisualSelect.IsSelecting {
			m.VisualSelect.IsSelecting = false
			m.Chat.Viewport.SetContent(m.renderConversation())
			return m, nil, true
		}
		return m.exitVisualMode()

	case tea.KeyEnter:
		block := m.getCurrentBlock()
		switch m.VisualSelect.Mode {
		case visualModeGenerate:
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
				if m.Chat.IsStreaming {
					m.Session.CancelGeneration()
					m.Chat.IsStreaming = false
					m.Chat.StreamSub = nil
				}

				// Exit visual mode before starting generation
				m.State = stateIdle
				m.VisualSelect.Mode = visualModeNone
				m.Chat.TextArea.Focus()

				event := m.Session.RegenerateFrom(msgIndex)
				model, cmd := m.startGeneration(event)
				return model, cmd, true
			}
			// If no user message found (should be impossible if there are blocks),
			// fall through to exit visual mode.

		case visualModeEdit:
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
				m.VisualSelect.Mode = visualModeNone
				m.Chat.EditingMessageIndex = userMsgIndex
				originalContent := m.Session.GetMessages()[userMsgIndex].Content
				return m, editInEditorCmd(originalContent), true
			}
			// Fall through to exit visual mode if no user message found.
		case visualModeBranch:
			endMessageIndex := block.endIdx

			if m.Chat.IsStreaming {
				m.Session.CancelGeneration()
				m.Chat.IsStreaming = false
				m.Chat.StreamSub = nil
			}

			newSess, err := m.Session.Branch(endMessageIndex)
			if err != nil {
				m.StatusBarMessage = fmt.Sprintf("Error branching session: %v", err)
				cmd = clearStatusBarCmd()
			} else {
				m.Session = newSess
				m.addActiveSession(newSess)
				m.StatusBarMessage = "Branched to a new session."
				cmd = clearStatusBarCmd()

				// Exit visual mode and apply changes
				m.State = stateIdle
				m.VisualSelect.Mode = visualModeNone
				m.VisualSelect.IsSelecting = false

				// Reset UI state for new session
				m.Chat.LastInteractionFailed = false
				m.Chat.LastRenderedAIPart = ""
				m.Chat.TextArea.Reset()
				m.Chat.TextArea.SetHeight(1)
				m.Chat.TextArea.Focus()
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.GotoBottom()

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
		m.VisualSelect.Mode = visualModeNone
		m.VisualSelect.IsSelecting = false
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m, tea.Batch(textarea.Blink, cmd, tea.Batch(cmds...)), true

	case tea.KeyRunes:
		switch string(msg.Runes) {
		case km.VisualMode.Down:
			if m.VisualSelect.Cursor < len(m.VisualSelect.Blocks)-1 {
				m.VisualSelect.Cursor++
				offset := m.Chat.Viewport.YOffset
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.SetYOffset(offset)
			}
			// Allow the viewport to handle the key press for scrolling
			return m, nil, false
		case km.VisualMode.Up:
			if m.VisualSelect.Cursor > 0 {
				m.VisualSelect.Cursor--
				offset := m.Chat.Viewport.YOffset
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.SetYOffset(offset)
			}
			// Allow the viewport to handle the key press for scrolling
			return m, nil, false
		case km.VisualMode.Swap, strings.ToUpper(km.VisualMode.Swap):
			if m.VisualSelect.IsSelecting {
				m.VisualSelect.Cursor, m.VisualSelect.Start = m.VisualSelect.Start, m.VisualSelect.Cursor
				m.Chat.Viewport.SetContent(m.renderConversation())
			}
			return m, nil, true
		case km.VisualMode.Select, strings.ToUpper(km.VisualMode.Select):
			if m.VisualSelect.IsSelecting {
				m.VisualSelect.IsSelecting = false
			} else {
				m.VisualSelect.IsSelecting = true
				m.VisualSelect.Start = m.VisualSelect.Cursor
			}
			m.Chat.Viewport.SetContent(m.renderConversation())
			return m, nil, true
		case km.VisualMode.Branch:
			if !m.VisualSelect.IsSelecting && m.VisualSelect.Mode == visualModeNone {
				m.VisualSelect.Mode = visualModeBranch
				m.VisualSelect.IsSelecting = true // branch is a single-selection mode
				m.Chat.Viewport.SetContent(m.renderConversation())
				return m, nil, true
			}
		case km.VisualMode.New:
			if m.Chat.IsStreaming {
				m.Session.CancelGeneration()
				m.Chat.IsStreaming = false
				m.Chat.StreamSub = nil
			}
			event := m.Session.HandleShortcut(":new")
			if event.Type == types.NewSessionStarted {
				newModel, cmd := m.newSession(event.Mode)
				newModel.State = stateIdle
				newModel.VisualSelect.Mode = visualModeNone
				newModel.VisualSelect.IsSelecting = false
				return newModel, cmd, true
			}
			return m, nil, true
		case km.VisualMode.Exit:
			return m.exitVisualMode()
		case km.VisualMode.Regenerate:
			if !m.VisualSelect.IsSelecting && m.VisualSelect.Mode == visualModeNone {
				block := m.getCurrentBlock()
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
					if m.Chat.IsStreaming {
						m.Session.CancelGeneration()
						m.Chat.IsStreaming = false
						m.Chat.StreamSub = nil
					}

					// Exit visual mode before starting generation
					m.State = stateIdle
					m.VisualSelect.Mode = visualModeNone
					m.Chat.TextArea.Focus()

					event := m.Session.RegenerateFrom(msgIndex)
					model, cmd := m.startGeneration(event)
					return model, cmd, true
				}
			}
		case km.VisualMode.Edit:
			if !m.VisualSelect.IsSelecting && m.VisualSelect.Mode == visualModeNone {
				block := m.getCurrentBlock()
				userMsgIndex := -1
				// Find the first user message at or before the start of the selected block
				for i := block.startIdx; i >= 0; i-- {
					if m.Session.GetMessages()[i].Type == types.UserMessage {
						userMsgIndex = i
						break
					}
				}

				if userMsgIndex != -1 {
					m.Chat.EditingMessageIndex = userMsgIndex
					originalContent := m.Session.GetMessages()[userMsgIndex].Content
					return m, editInEditorCmd(originalContent), true
				}
			}
		case km.VisualMode.Copy:
			if m.VisualSelect.Mode == visualModeNone {
				indices := m.getSelectedIndices()
				if len(indices) == 0 {
					return m, nil, true
				}
				var selectedMessages []types.Message
				for _, idx := range indices {
					selectedMessages = append(selectedMessages, m.Session.GetMessages()[idx])
				}

				var content string
				if len(selectedMessages) == 1 && selectedMessages[0].Type == types.UserMessage {
					content = selectedMessages[0].Content
				} else {
					content = history.BuildHistorySnippet(selectedMessages)
				}

				cfg := m.Session.GetConfig()
				err := utils.Copy(content, cfg.Clipboard.CopyCmd)

				if err == nil {
					m.StatusBarMessage = "Copied to clipboard."
					cmd = clearStatusBarCmd()
				}
				if m.Chat.IsStreaming {
					messages := m.Session.GetMessages()
					if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
						m.State = stateThinking
					} else {
						m.State = stateGenerating
					}
					cmd = tea.Batch(cmd, m.Chat.Spinner.Tick)
				} else {
					m.State = stateIdle
				}
				m.VisualSelect.Mode = visualModeNone
				m.VisualSelect.IsSelecting = false
				m.Chat.TextArea.Focus()
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.GotoBottom()
				return m, tea.Batch(textarea.Blink, cmd), true
			}
		case km.VisualMode.Delete:
			if m.VisualSelect.Mode == visualModeNone {
				selectedIndices := m.getSelectedIndices()
				if len(selectedIndices) == 0 {
					return m, nil, true
				}

				isDeletingCurrentAIMessage := false
				if m.Chat.IsStreaming && len(m.Session.GetMessages()) > 0 {
					lastMessageIndex := len(m.Session.GetMessages()) - 1

					if slices.Contains(selectedIndices, lastMessageIndex) {
						isDeletingCurrentAIMessage = true
					}
				}
				if isDeletingCurrentAIMessage {
					m.Session.CancelGeneration()
					m.Chat.IsStreaming = false
					m.Chat.StreamSub = nil
				}
				m.Session.DeleteMessages(selectedIndices)
				m.ClearCache()
				m.StatusBarMessage = "Deleted selected messages."
				cmd = clearStatusBarCmd()
				if m.Chat.IsStreaming {
					messages := m.Session.GetMessages()
					if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
						m.State = stateThinking
					} else {
						m.State = stateGenerating
					}
					cmd = tea.Batch(cmd, m.Chat.Spinner.Tick)
				} else {
					m.State = stateIdle
				}
				m.VisualSelect.Mode = visualModeNone
				m.VisualSelect.IsSelecting = false
				m.Chat.TextArea.Focus()
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.GotoBottom()
				m.IsCountingTokens = true
				return m, tea.Batch(textarea.Blink, cmd, countTokensCmd(m.Session.GetPrompt())), true
			}
		}
	}
	return m, nil, true
}

func (m Model) getSelectedIndices() []int {
	var indices []int
	if m.VisualSelect.IsSelecting {
		start, end := m.VisualSelect.Start, m.VisualSelect.Cursor
		if start > end {
			start, end = end, start
		}
		for i := start; i <= end; i++ {
			if i < len(m.VisualSelect.Blocks) {
				block := m.VisualSelect.Blocks[i]
				for j := block.startIdx; j <= block.endIdx; j++ {
					indices = append(indices, j)
				}
			}
		}
	} else if m.VisualSelect.Cursor < len(m.VisualSelect.Blocks) {
		block := m.VisualSelect.Blocks[m.VisualSelect.Cursor]
		for j := block.startIdx; j <= block.endIdx; j++ {
			indices = append(indices, j)
		}
	}
	return indices
}

func (m Model) getCurrentBlock() messageBlock {
	if len(m.VisualSelect.Blocks) == 0 {
		return messageBlock{startIdx: -1, endIdx: -1}
	}
	if m.VisualSelect.Cursor >= len(m.VisualSelect.Blocks) {
		return m.VisualSelect.Blocks[len(m.VisualSelect.Blocks)-1]
	}
	return m.VisualSelect.Blocks[m.VisualSelect.Cursor]
}

func (m Model) exitVisualMode() (Model, tea.Cmd, bool) {
	var cmd tea.Cmd = textarea.Blink
	if m.Chat.IsStreaming {
		messages := m.Session.GetMessages()
		// Check if the last message is an empty AI message, which indicates 'thinking' state.
		if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
			m.State = stateThinking
		} else {
			m.State = stateGenerating
		}
		cmd = tea.Batch(cmd, m.Chat.Spinner.Tick)
	} else {
		m.State = stateIdle
	}
	m.VisualSelect.Mode = visualModeNone
	m.VisualSelect.IsSelecting = false
	m.Chat.TextArea.Focus()
	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()
	return m, cmd, true
}
