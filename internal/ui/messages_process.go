package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/types"
)

func (m Model) handleProcessMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case editorFinishedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Editor Error:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			m.Chat.EditingMessageIndex = -1
			return m, tea.EnableMouseCellMotion, true
		}

		if m.Chat.EditingMessageIndex != -1 {
			if msg.content != msg.originalContent {
				if err := m.Session.EditMessage(m.Chat.EditingMessageIndex, msg.content); err != nil {
					errorContent := fmt.Sprintf("\n**Editor Error:**\n```\nFailed to apply edit: %v\n```\n", err)
					m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
				}
			}

			var cmd tea.Cmd
			if m.Chat.IsStreaming {
				messages := m.Session.GetMessages()
				if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
					m.State = stateAsking
				} else {
					m.State = stateGenerating
				}
				cmd = m.Chat.Spinner.Tick
			} else {
				m.State = stateIdle
				cmd = textarea.Blink
			}

			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()

			m.Chat.EditingMessageIndex = -1
			return m, tea.Batch(cmd, tea.EnableMouseCellMotion, m.updateTokenCountCmd(), m.renderUncachedCmd()), true
		}

		if msg.content != msg.originalContent {
			m.Chat.TextArea.SetValue(msg.content)
			m.Chat.TextArea.CursorEnd()
			model, cmd := m.handleSubmit()
			return model, tea.Batch(cmd, tea.EnableMouseCellMotion), true
		}

		m.Chat.TextArea.SetValue(msg.originalContent)
		m.Chat.TextArea.Focus()
		return m, tea.Batch(textarea.Blink, tea.EnableMouseCellMotion), true

	case fileEditorFinishedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Editor Error:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			return m, tea.EnableMouseCellMotion, true
		}
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
			return m, tea.Batch(textarea.Blink, tea.EnableMouseCellMotion), true
		}
		return m, nil, true

	case shellFinishedMsg:
		if msg.cmdStr != "" {
			resType := types.ShellCmdResultMessage
			content := msg.output
			if msg.err != nil && content == "" {
				resType = types.CommandErrorResultMessage
				content = fmt.Sprintf("Command failed: %v", msg.err)
			} else if content == "" {
				content = "Command completed with no output."
			}
			m.Session.AddMessages(types.Message{
				Type:    resType,
				Content: content,
			})
		}
		m.State = stateIdle
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.Chat.TextArea.Reset()
		return m, tea.Batch(textarea.Blink, tea.EnableMouseCellMotion, m.updateTokenCountCmd()), true
	}

	return m, nil, false
}
