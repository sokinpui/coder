package ui

import (
	"coder/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) startGeneration(event core.Event) (Model, tea.Cmd) {
	if event.Type != core.GenerationStarted {
		return m, nil // Should not happen
	}
	m.state = stateThinking
	m.isStreaming = true
	m.streamSub = event.Data.(chan string)
	m.textArea.Blur()
	m.textArea.Reset()

	m.lastRenderedAIPart = ""
	m.lastInteractionFailed = false

	m.viewport.SetContent(m.renderConversation())
	m.viewport.GotoBottom()

	prompt := m.session.GetPromptForTokenCount()
	m.isCountingTokens = true
	return m, tea.Batch(listenForStream(m.streamSub), m.spinner.Tick, countTokensCmd(prompt))
}

func (m Model) handleKeyPressGenerating(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
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
		if event.Type == core.NewSessionStarted {
			newModel, cmd := m.newSession()
			newModel.state = stateIdle
			return newModel, cmd, true
		}
		return m, nil, true
	case tea.KeyCtrlB:
		event := m.session.HandleInput(":branch")
		if event.Type == core.BranchModeStarted {
			model, cmd := m.enterVisualMode(visualModeBranch)
			return model, cmd, true
		} else if event.Type == core.MessagesUpdated {
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
		m, _ = m.enterVisualMode(visualModeNone)
	}
	return m, nil, true
}
