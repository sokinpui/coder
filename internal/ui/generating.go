package ui

import (
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/commands"
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
	if m.shouldFocusTextArea() {
		m.Chat.TextArea.Focus()
	} else {
		m.Chat.TextArea.Blur()
	}
	m = m.updateLayout()

	m.Chat.LastInteractionFailed = false

	messages := m.Session.GetMessages()
	if len(messages) > 0 {
		delete(m.Chat.RenderCache, len(messages)-1)
	}

	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()

	cmds := []tea.Cmd{
		listenForEvents(m.Session.GetID(), m.Chat.EventSub),
		m.Chat.Spinner.Tick,
		m.renderUncachedCmd(),
		m.updateTokenCountCmd(),
	}
	if m.shouldFocusTextArea() {
		cmds = append(cmds, textarea.Blink)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) cancelGeneration() (Model, tea.Cmd) {
	if m.ConfirmRequest != nil {
		if approver, ok := m.Session.(engine.ToolApprover); ok {
			_ = approver.RespondToolConfirmation(m.ConfirmRequest.CallID, types.ToolConfirmResponse{Approved: false})
		}
		m.ConfirmRequest = nil
	}
	if m.Btw.Active {
		m.Btw.reset()
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
			m.Session.AddMessages(types.Message{Type: types.InterruptedMessage, Content: "Interrupted by user."})
		} else if lastMsg.Type == types.AIMessage {
			lastMsg.Content = "Interrupted by user."
			lastMsg.Type = types.InterruptedMessage
			m.Session.ReplaceLastMessage(lastMsg)
		} else {
			m.Session.AddMessages(types.Message{Type: types.InterruptedMessage, Content: "Interrupted by user."})
		}
	}

	m.Chat.TextArea.Focus()
	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()
	m = m.updateLayout()
	return m, tea.Batch(textarea.Blink, m.updateTokenCountCmd(), m.renderUncachedCmd(), saveConversationCmd(m.Session))
}

func (m Model) handleKeyPressGenerating(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	keyStr := msg.String()
	km := m.Keymap()

	switch msg.Type {
	case tea.KeyCtrlC:
		newModel, cmd := m.cancelGeneration()
		return newModel, cmd, true
	case tea.KeyCtrlN:
		newModel, cmd := m.newSession("")
		newModel.State = stateIdle
		return newModel, cmd, true
	case tea.KeyEscape:
		newModel, cmd := m.openAtomicMsgMode()
		return newModel, cmd, true
	}

	if m.canInputWhileStreaming() && msg.Type == tea.KeyEnter {
		input := m.Chat.TextArea.Value()
		if commands.IsCommand(input) {
			if cmdName, args, isCmd := commands.ParseCommand(input); isCmd && cmdName == "queue" {
				_ = args
				m.Chat.TextArea.Reset()
				m = m.updateLayout()
				newModel, cmd := m.openQueueSelector()
				return newModel, cmd, true
			} else if isCmd && cmdName == "btw" {
				m.Chat.TextArea.Reset()
				m = m.updateLayout()
				newModel, cmd, _ := m.openBtwOverlay(args)
				return newModel, cmd, true
			}
			m.enqueuePrompt(input)
			m.Chat.TextArea.Reset()
			m = m.updateLayout()
			return m, nil, true
		}
		return m, nil, false
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

	if m.canInputWhileStreaming() {
		switch keyStr {
		case km.Submit:
			input := m.Chat.TextArea.Value()
			if strings.TrimSpace(input) == "" {
				return m, nil, true
			}
			if cmdName, args, isCmd := commands.ParseCommand(input); isCmd && cmdName == "queue" {
				m.Chat.TextArea.Reset()
				m = m.updateLayout()
				newModel, cmd := m.openQueueSelector()
				return newModel, cmd, true
			} else if isCmd && cmdName == "btw" {
				m.Chat.TextArea.Reset()
				m = m.updateLayout()
				newModel, cmd, _ := m.openBtwOverlay(args)
				return newModel, cmd, true
			}
			m.enqueuePrompt(input)
			m.Chat.TextArea.Reset()
			m = m.updateLayout()
			return m, nil, true
		case km.Editor:
			if m.Chat.TextArea.Focused() {
				return m, editInEditorCmd(m.Chat.TextArea.Value()), true
			}
		case km.Paste:
			m.Chat.IsPastingImage = true
			return m, tea.Batch(
				HandlePasteCmd(m.Session.GetConfig(), m.KittyImage, m.Chat.Viewport.Width),
				m.Chat.Spinner.Tick,
			), true
		}
		return m, nil, false
	}

	return m, nil, true
}
