package update

import (
	"coder/internal/core"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKeyPressGenPending(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.State = stateIdle
		m.TextArea.Focus()
		m.Session.AddMessage(core.Message{
			Type:    core.CommandResultMessage, // Re-use style for notification
			Content: "Generation cancelled.",
		})
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()
		return m, textarea.Blink, true
	}
	return m, nil, true // Consume all key presses
}

func (m Model) startGeneration(event core.Event) (Model, tea.Cmd) {
	if event.Type != core.GenerationStarted {
		return m, nil // Should not happen
	}
	m.State = stateThinking
	m.IsStreaming = true
	m.StreamSub = event.Data.(chan string)
	m.TextArea.Blur()
	m.TextArea.Reset()

	m.LastRenderedAIPart = ""
	m.LastInteractionFailed = false

	m.Viewport.SetContent(m.renderConversation())
	m.Viewport.GotoBottom()

	prompt := m.Session.GetPrompt()
	m.IsCountingTokens = true
	return m, tea.Batch(listenForStream(m.StreamSub), m.Spinner.Tick, countTokensCmd(prompt))
}

func (m Model) handleKeyPressGenerating(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.State != stateCancelling {
			m.Session.CancelGeneration()
			m.State = stateCancelling
		}
	case tea.KeyCtrlN:
		if m.IsStreaming {
			m.Session.CancelGeneration()
			m.IsStreaming = false
			m.StreamSub = nil
		}
		event := m.Session.HandleInput(":new")
		if event.Type == core.NewSessionStarted {
			newModel, cmd := m.newSession()
			newModel.State = stateIdle
			return newModel, cmd, true
		}
		return m, nil, true
	case tea.KeyCtrlB:
		event := m.Session.HandleInput(":branch")
		if event.Type == core.BranchModeStarted {
			model, cmd := m.enterVisualMode(visualModeBranch)
			return model, cmd, true
		} else if event.Type == core.MessagesUpdated {
			// This handles the case where branching is not possible (e.g., no messages)
			// and an error message was added to the session.
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
		}
		return m, nil, true
	case tea.KeyCtrlH:
		m.State = stateHistorySelect
		m.TextArea.Blur()
		return m, listHistoryCmd(m.Session.GetHistoryManager()), true
	case tea.KeyEscape:
		// Allow entering visual mode even during generation.
		m, _ = m.enterVisualMode(visualModeNone)
	}
	return m, nil, true
}
