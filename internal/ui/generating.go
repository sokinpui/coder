package ui

import (
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) startGenerationEvents(eventChan <-chan types.SessionEvent) (Model, tea.Cmd) {
	m.State = stateAsking
	m.Chat.StateStartTime = time.Now()
	m.Chat.IsStreaming = true
	m.Chat.IsAIRendering = false
	m.Chat.PendingAIRender = false
	m.Chat.EventSub = eventChan
	m.Chat.AutoScroll = true
	m.Chat.TextArea.Blur()
	m.Chat.TextArea.Reset()
	m = m.updateLayout()

	m.Chat.LastInteractionFailed = false

	messages := m.Session.GetMessages()
	if len(messages) > 0 {
		delete(m.Chat.RenderCache, len(messages)-1)
	}

	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()

	return m, tea.Batch(listenForEvents(m.Session.GetID(), m.Chat.EventSub), m.Chat.Spinner.Tick, m.renderUncachedCmd())
}

func (m Model) handleKeyPressGenerating(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	keyStr := msg.String()
	km := m.Keymap()

	switch msg.Type {
	case tea.KeyCtrlC:
		if m.ConfirmRequest != nil {
			if approver, ok := m.Session.(engine.ToolApprover); ok {
				_ = approver.RespondToolConfirmation(m.ConfirmRequest.CallID, types.ToolConfirmResponse{Approved: false})
			}
			m.ConfirmRequest = nil
		}
		m.ActiveOverlay = overlayNone
		m.Session.Cancel()
		m.Chat.IsStreaming = false
		m.Chat.IsAIRendering = false
		m.Chat.PendingAIRender = false
		m.Chat.EventSub = nil
		m.Chat.LastInteractionFailed = true
		m.StatusText = ""
		m.State = stateIdle

		messages := m.Session.GetMessages()
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if lastMsg.Type == types.AIMessage && strings.TrimSpace(lastMsg.Content) != "" {
				m.Session.AddMessages(types.Message{Type: types.CommandResultMessage, Content: "Generation cancelled."})
			} else if lastMsg.Type == types.AIMessage {
				lastMsg.Content = "Generation cancelled."
				lastMsg.Type = types.CommandResultMessage
				m.Session.ReplaceLastMessage(lastMsg)
			} else {
				m.Session.AddMessages(types.Message{Type: types.CommandResultMessage, Content: "Generation cancelled."})
			}
		}

		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m = m.updateLayout()
		return m, textarea.Blink, true
	case tea.KeyCtrlN:
		newModel, cmd := m.newSession("")
		newModel.State = stateIdle
		return newModel, cmd, true
	case tea.KeyEscape:
		newModel, cmd := m.openAtomicMsgMode()
		return newModel, cmd, true
	}

	if keyStr == km.Msg || keyStr == "esc" {
		newModel, cmd := m.openAtomicMsgMode()
		return newModel, cmd, true
	}

	switch keyStr {
	case km.New:
		newModel, cmd := m.newSession("")
		newModel.State = stateIdle
		return newModel, cmd, true
	case km.Branch:
		model, cmd := m.openAtomicMsgMode()
		return model, cmd, true
	case km.History:
		newModel, cmd := m.openHistorySelector(0)
		return newModel, cmd, true
	}
	return m, nil, true
}
