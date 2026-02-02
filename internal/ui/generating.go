package ui

import (
	"github.com/sokinpui/coder/internal/types"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPressGenPending(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.State = stateIdle
		m.Chat.StreamBuffer = ""
		m.Chat.IsStreamAnime = false
		m.Chat.StreamDone = false
		m.Chat.TextArea.Focus()
		m.Session.AddMessages(types.Message{
			Type:    types.CommandResultMessage, // Re-use style for notification
			Content: "Generation cancelled.",
		})
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m, textarea.Blink, true
	}
	return m, nil, true // Consume all key presses
}

func (m Model) startGeneration(event types.Event) (Model, tea.Cmd) {
	if event.Type != types.GenerationStarted {
		return m, nil // Should not happen
	}
	m.State = stateThinking
	m.Chat.IsStreaming = true
	m.Chat.StreamBuffer = ""
	m.Chat.StreamDone = false
	m.Chat.IsStreamAnime = false
	m.Chat.StreamSub = event.Data.(chan string)
	m.Chat.TextArea.Blur()
	m.Chat.TextArea.Reset()
	m = m.updateLayout()

	m.Chat.LastRenderedAIPart = ""
	m.Chat.LastInteractionFailed = false

	m.Chat.Viewport.SetContent(m.renderConversation())
	m.Chat.Viewport.GotoBottom()

	prompt := m.Session.GetPrompt()
	m.IsCountingTokens = true
	return m, tea.Batch(listenForStream(m.Chat.StreamSub), m.Chat.Spinner.Tick, countTokensCmd(prompt))
}

func (m Model) handleKeyPressGenerating(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.State != stateCancelling {
			m.Session.CancelGeneration()
			m.State = stateCancelling
		}
	case tea.KeyCtrlN:
		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false
			m.Chat.StreamSub = nil
		}
		event := m.Session.HandleInput(":new")
		switch event.Type {
		case types.NewSessionStarted:
			newModel, cmd := m.newSession()
			newModel.State = stateIdle
			return newModel, cmd, true
		}
		return m, nil, true
	case tea.KeyCtrlB:
		event := m.Session.HandleInput(":branch")
		switch event.Type {
		case types.BranchModeStarted:
			model, cmd := m.enterVisualMode(visualModeBranch)
			return model, cmd, true
		case types.MessagesUpdated:
			// This handles the case where branching is not possible (e.g., no messages)
			// and an error message was added to the session.
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
		}
		return m, nil, true
	case tea.KeyCtrlH:
		m.State = stateHistorySelect
		m.Chat.TextArea.Blur()
		return m, listHistoryCmd(m.Session.GetHistoryManager()), true
	case tea.KeyEscape:
		// Allow entering visual mode even during generation.
		m, _ = m.enterVisualMode(visualModeNone)
	}
	return m, nil, true
}
