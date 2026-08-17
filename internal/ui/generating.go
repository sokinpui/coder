package ui

import (
	"time"

	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) startGeneration(event types.Event) (Model, tea.Cmd) {
	if event.Type != types.GenerationStarted {
		return m, nil // Should not happen
	}
	m.State = stateAsking
	m.Chat.StateStartTime = time.Now()
	m.Chat.IsStreaming = true
	m.Chat.StreamSub = event.Data.(chan types.StreamChunk)
	m.Chat.TextArea.Blur()
	m.Chat.TextArea.Reset()
	m = m.updateLayout()

	m.Chat.LastInteractionFailed = false

	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()

	return m, tea.Batch(listenForStream(m.Chat.StreamSub), m.Chat.Spinner.Tick)
}

func (m Model) handleKeyPressGenerating(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	keyStr := msg.String()
	km := m.Session.GetConfig().Keymap

	switch msg.Type {
	case tea.KeyCtrlC:
		if m.State != stateCancelling {
			m.Session.CancelGeneration()
			m.State = stateCancelling
			m.Chat.StateStartTime = time.Now()
		} else {
			// Emergency cancel: force return to idle if already cancelling
			m.State = stateIdle
			m.Chat.IsStreaming = false
			m.Chat.LastInteractionFailed = true
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
	case tea.KeyCtrlN:
		event := m.Session.HandleInput("/new")
		if event.Type != types.NewSessionStarted {
			return m, nil, true
		}
		newModel, cmd := m.newSession(event.Mode)
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
		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false
			m.Chat.StreamSub = nil
		}
		event := m.Session.HandleInput("/new")
		switch event.Type {
		case types.NewSessionStarted:
			newModel, cmd := m.newSession(event.Mode)
			newModel.State = stateIdle
			return newModel, cmd, true
		}
		return m, nil, true
	case km.Branch:
		event := m.Session.HandleInput("/branch")
		switch event.Type {
		case types.BranchModeStarted, types.AtomicMsgModeStarted:
			model, cmd := m.openAtomicMsgMode()
			return model, cmd, true
		case types.MessagesUpdated:
			// This handles the case where branching is not possible (e.g., no messages)
			// and an error message was added to the session.
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
		}
		return m, nil, true
	case km.History:
		m.State = stateHistorySelect
		m.Chat.TextArea.Blur()
		m = m.updateLayout()
		return m, listHistoryCmd(m.Session.GetHistoryManager()), true
	}
	return m, nil, true
}
