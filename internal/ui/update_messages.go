package ui

import (
	"fmt"
	"time"

	"coder/internal/core"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		// Tick the spinner during all generation phases.
		if m.state != stateThinking && m.state != stateGenerating && m.state != stateCancelling {
			return m, nil, true
		}

		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)

		// If we are in the "thinking" state, the spinner is in the viewport.
		// We need to update the viewport's content to reflect the spinner's animation.
		if m.state == stateThinking {
			m.viewport.SetContent(m.renderConversation())
		}
		return m, spinnerCmd, true

	case streamResultMsg:
		messages := m.session.GetMessages()
		lastMsg := messages[len(messages)-1]
		if m.state == stateThinking {
			m.state = stateGenerating
			lastMsg.Content += string(msg)
			m.session.ReplaceLastMessage(lastMsg)
			return m, tea.Batch(listenForStream(m.streamSub), renderTick()), true
		}
		lastMsg.Content += string(msg)
		m.session.ReplaceLastMessage(lastMsg)
		return m, listenForStream(m.streamSub), true

	case streamFinishedMsg:
		m.isStreaming = false
		messages := m.session.GetMessages()

		if m.state == stateCancelling {
			// This was a cancellation.
			lastMsg := messages[len(messages)-1]
			lastMsg.Content = "Generation cancelled."
			lastMsg.Type = core.CommandResultMessage // Re-use style for notification
			m.session.ReplaceLastMessage(lastMsg)
			m.lastInteractionFailed = true
		}

		wasAtBottom := m.viewport.AtBottom()
		m.viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.viewport.GotoBottom()
		}

		m.state = stateIdle
		m.streamSub = nil
		m.session.CancelGeneration()
		m.textArea.Reset()
		m.textArea.Focus()

		if m.lastInteractionFailed {
			return m, nil, true // Don't count tokens on failure/cancellation
		}

		prompt := m.session.GetPromptForTokenCount()
		m.isCountingTokens = true

		return m, countTokensCmd(prompt), true

	case editorFinishedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Editor Error:**\n```\n%v\n```\n", msg.err)
			m.session.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: errorContent})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			m.editingMessageIndex = -1 // Also reset here
			return m, nil, true
		}

		if m.editingMessageIndex != -1 {
			// This block handles the return from editing a previous message in the history.
			// It updates the message in place and does not trigger a new generation.
			if err := m.session.EditMessage(m.editingMessageIndex, msg.content); err != nil {
				// This should ideally not happen if the logic for selecting an editable message is correct.
				errorContent := fmt.Sprintf("\n**Editor Error:**\n```\nFailed to apply edit: %v\n```\n", err)
				m.session.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: errorContent})
			}

			m.textArea.Focus()
			m.viewport.SetContent(m.renderConversation())

			m.editingMessageIndex = -1 // Reset on success or failure
			return m, textarea.Blink, true
		}

		// This is for Ctrl+E on the text area.
		m.textArea.SetValue(msg.content)
		m.textArea.CursorEnd()
		m.textArea.Focus()
		return m, nil, true

	case renderTickMsg:
		if m.state != stateGenerating || !m.isStreaming {
			return m, nil, true
		}

		messages := m.session.GetMessages()
		lastMsg := messages[len(messages)-1]
		if lastMsg.Content != m.lastRenderedAIPart {
			wasAtBottom := m.viewport.AtBottom()
			m.viewport.SetContent(m.renderConversation())
			if wasAtBottom {
				m.viewport.GotoBottom()
			}
			m.lastRenderedAIPart = lastMsg.Content
		}

		return m, renderTick(), true

	case tokenCountResultMsg:
		m.tokenCount = int(msg)
		m.isCountingTokens = false
		return m, nil, true

	case historyListResultMsg:
		if msg.err != nil {
			m.statusBarMessage = fmt.Sprintf("Error loading history: %v", msg.err)
			m.state = stateIdle
			m.textArea.Focus()
			return m, tea.Batch(clearStatusBarCmd(5*time.Second), textarea.Blink), true
		}
		m.historyItems = msg.items
		m.historySelectCursor = 0
		m.viewport.SetContent(m.historyView())
		m.viewport.GotoTop()
		return m, nil, true

	case conversationLoadedMsg:
		if msg.err != nil {
			m.statusBarMessage = fmt.Sprintf("Error loading conversation: %v", msg.err)
			m.state = stateIdle
			m.textArea.Focus()
			return m, tea.Batch(clearStatusBarCmd(5*time.Second), textarea.Blink), true
		}
		m.state = stateIdle
		m.lastInteractionFailed = false
		m.lastRenderedAIPart = ""
		m.textArea.Reset()
		m.textArea.SetHeight(1)
		m.textArea.Focus()
		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()
		m.isCountingTokens = true
		return m, tea.Batch(countTokensCmd(m.session.GetPromptForTokenCount()), textarea.Blink), true

	case titleGeneratedMsg:
		// The title is already updated in the session.
		// This message just triggers a re-render of the status bar.
		return m, nil, true

	case clearStatusBarMsg:
		m.statusBarMessage = ""
		return m, nil, true

	case ctrlCTimeoutMsg:
		m.ctrlCPressed = false
		return m, nil, true

	case initialContextLoadedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Error loading initial context:**\n```\n%v\n```\n", msg.err)
			m.session.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: errorContent})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil, true
		}

		// Now that context is loaded, count the tokens.
		m.isCountingTokens = true
		return m, countTokensCmd(m.session.GetInitialPromptForTokenCount()), true

	case errorMsg:
		m.isStreaming = false

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		messages := m.session.GetMessages()
		lastMsg := messages[len(messages)-1]
		lastMsg.Content = errorContent
		m.session.ReplaceLastMessage(lastMsg)
		m.lastInteractionFailed = true

		wasAtBottom := m.viewport.AtBottom()
		m.viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.viewport.GotoBottom()
		}
		m.state = stateIdle
		m.streamSub = nil
		m.session.CancelGeneration()
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil, true

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.textArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalPadding())
		m.viewport.Width = msg.Width

		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.viewport.Width),
		)
		if err == nil {
			m.glamourRenderer = renderer
			m.viewport.SetContent(m.renderConversation())
		}
		return m, nil, false
	}
	return m, nil, false
}
