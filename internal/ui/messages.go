package ui

import (
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case modelsFetchedMsg:
		m.Chat.IsFetchingModels = false
		if msg.err != nil {
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: fmt.Sprintf("Failed to fetch models: %v", msg.err),
			})
			if m.ActiveOverlay == overlayNone {
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.GotoBottom()
			}
			return m, nil, true
		}

		cfg := m.Session.GetConfig()
		cfg.AvailableModels = msg.models

		if len(msg.models) == 0 {
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: "Warning: Server returned no available models.",
			})
			if m.ActiveOverlay == overlayNone {
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.GotoBottom()
			}
			return m, nil, true
		}

		// Validation
		hasError := false
		var errorStrings []string
		if !slices.Contains(msg.models, cfg.Generation.ModelCode) {
			errorStrings = append(errorStrings, fmt.Sprintf("Configured chat model '%s' is not in the available list.", cfg.Generation.ModelCode))
			hasError = true
		}
		if !slices.Contains(msg.models, cfg.Generation.TitleModelCode) {
			errorStrings = append(errorStrings, fmt.Sprintf("Configured title model '%s' is not in the available list.", cfg.Generation.TitleModelCode))
			hasError = true
		}

		if hasError {
			errorStrings = append(errorStrings, fmt.Sprintf("Available models: %v", msg.models))
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: strings.Join(errorStrings, "\n"),
			})
			if m.ActiveOverlay == overlayNone {
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.GotoBottom()
			}
		}
		return m, nil, true

	case spinner.TickMsg:
		if !m.needsSpinner() {
			return m, nil, true
		}

		var spinnerCmd tea.Cmd
		m.Chat.Spinner, spinnerCmd = m.Chat.Spinner.Update(msg)
		if spinnerCmd == nil {
			spinnerCmd = m.Chat.Spinner.Tick
		}

		// We need to update the viewport's content to reflect the spinner's animation.
		switch m.State {
		case stateAsking, stateThinking:
			wasAtBottom := m.Chat.Viewport.AtBottom()
			m.Chat.Viewport.SetContent(m.renderConversation())
			if wasAtBottom {
				m.Chat.Viewport.GotoBottom()
			}
		}
		return m, spinnerCmd, true

	case streamResultMsg:
		if !m.Chat.IsStreaming {
			return m, nil, true
		}

		if msg.ReasoningContent != "" && m.State != stateGenerating {
			m.State = stateThinking
		}
		if msg.Content != "" {
			if m.State != stateGenerating {
				m.State = stateGenerating
				m.Chat.StateStartTime = time.Now()
			}
			messages := m.Session.GetMessages()
			if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage {
				messages[len(messages)-1].Content += msg.Content
			} else {
				m.Session.AddMessages(types.Message{Type: types.AIMessage, Content: msg.Content})
			}

			wasAtBottom := m.Chat.Viewport.AtBottom()
			m.Chat.Viewport.SetContent(m.renderConversation())
			if wasAtBottom {
				m.Chat.Viewport.GotoBottom()
			}
		}

		return m, listenForStream(m.Chat.StreamSub), true

	case streamFinishedMsg:
		if !m.Chat.IsStreaming {
			return m, nil, true
		}

		m.Chat.IsStreaming = false

		messages := m.Session.GetMessages()

		switch m.State {
		case stateCancelling:
			// This was a cancellation.
			if len(messages) > 0 {
				lastMsg := messages[len(messages)-1]
				if lastMsg.Type == types.AIMessage && strings.TrimSpace(lastMsg.Content) != "" {
					m.Session.AddMessages(types.Message{Type: types.CommandResultMessage, Content: "Generation cancelled."})
				} else {
					lastMsg.Content = "Generation cancelled."
					lastMsg.Type = types.CommandResultMessage
					m.Session.ReplaceLastMessage(lastMsg)
				}
			}
			m.Chat.LastInteractionFailed = true
		}

		m.State = stateIdle
		if m.ActiveOverlay == overlayNone {
			m.Chat.TextArea.Focus()
		}
		wasAtBottom := m.Chat.Viewport.AtBottom()
		m.Chat.Viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.Chat.Viewport.GotoBottom()
		}

		m.Chat.StreamSub = nil
		m.Session.CancelGeneration()
		m.Chat.TextArea.Reset()
		m = m.updateLayout()

		if m.Chat.LastInteractionFailed {
			return m, nil, true // Don't count tokens on failure/cancellation
		}
		m.UpdateTokenCount()

		return m, tea.Batch(saveConversationCmd(m.Session), m.Chat.Spinner.Tick), true

	case editorFinishedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Editor Error:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			m.Chat.EditingMessageIndex = -1 // Also reset here
			return m, nil, true
		}

		if m.Chat.EditingMessageIndex != -1 {
			// This block handles the return from editing a previous message in the history.
			// It updates the message in place and does not trigger a new generation.
			if msg.content != msg.originalContent {
				if err := m.Session.EditMessage(m.Chat.EditingMessageIndex, msg.content); err != nil {
					// This should ideally not happen if the logic for selecting an editable message is correct.
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

			m.Chat.EditingMessageIndex = -1 // Reset on success or failure
			m.UpdateTokenCount()
			return m, cmd, true
		}

		// This is for Ctrl+E on the text area. If content changed, submit.
		if msg.content != msg.originalContent {
			m.Chat.TextArea.SetValue(msg.content)
			m.Chat.TextArea.CursorEnd()
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}

		// Content is unchanged, just update textarea and focus.
		m.Chat.TextArea.SetValue(msg.originalContent)
		m.Chat.TextArea.Focus()
		return m, textarea.Blink, true

	case fileEditorFinishedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Editor Error:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			return m, nil, true
		}
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true

	case historyListResultMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading history: %v", msg.err)
			if m.ActiveOverlay == overlayHistory {
				m.ActiveOverlay = overlayNone
			}
			if m.State == stateIdle && m.ActiveOverlay == overlayNone {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}
		m.History.Items = msg.items
		m.History.FilteredItems = msg.items

		currentFilename := m.Session.GetHistoryFilename()
		initialCursorPos := 0
		if currentFilename != "" {
			for i, item := range msg.items {
				if item.Filename == currentFilename {
					initialCursorPos = i
					break
				}
			}
		}
		m.History.CursorPos = initialCursorPos
		return m, nil, true

	case conversationLoadedMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading conversation: %v", msg.err)
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		if m.Session != nil {
			if err := m.Session.SaveConversation(); err != nil {
				log.Printf("Error saving current conversation before switching: %v", err)
			}
		}

		m.ActiveOverlay = overlayNone
		m.Session = msg.sess
		m.ClearCache()
		m.addActiveSession(msg.sess)

		welcome := types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage}
		dirInfo := types.Message{Type: types.DirectoryMessage, Content: utils.GetDirInfoContent()}
		m.Session.PrependMessages(welcome, dirInfo)

		m.State = stateIdle
		m.Chat.LastInteractionFailed = false
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.SetHeight(1)
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.UpdateTokenCount()
		return m, textarea.Blink, true

	case switchActiveSessionMsg:
		if m.Session != nil {
			if err := m.Session.SaveConversation(); err != nil {
				log.Printf("Error saving current conversation before switching: %v", err)
			}
		}

		m.ActiveOverlay = overlayNone
		m.Session = msg.sess
		m.ClearCache()
		m.State = stateIdle
		m.Chat.LastInteractionFailed = false
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.SetHeight(1)
		m.Chat.TextArea.Focus()

		// Reload context to be safe
		if err := m.Session.LoadContext(); err != nil {
			log.Printf("Error reloading context for switched session: %v", err)
		}

		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.UpdateTokenCount()
		return m, textarea.Blink, true

	case titleGeneratedMsg:
		m.Chat.AnimatingTitle = true
		m.Chat.FullGeneratedTitle = msg.title
		m.Chat.DisplayedTitle = ""
		return m, animateTitleTick(), true

	case pasteResultMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Paste error: %v", msg.err)
			return m, clearStatusBarCmd(), true
		}

		if msg.isImage {
			m.Session.AddMessages(types.Message{Type: types.ImageMessage, Content: msg.content})
			m.UpdateTokenCount()
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
		} else {
			m.Chat.TextArea.InsertString(msg.content)
		}
		return m, nil, false

	case animateTitleTickMsg:
		if !m.Chat.AnimatingTitle {
			return m, nil, true
		}

		if len(m.Chat.DisplayedTitle) < len(m.Chat.FullGeneratedTitle) {
			// Use rune-safe slicing to handle multi-byte characters
			m.Chat.DisplayedTitle = string([]rune(m.Chat.FullGeneratedTitle)[:len([]rune(m.Chat.DisplayedTitle))+1])
			return m, animateTitleTick(), true
		}

		m.Chat.AnimatingTitle = false
		return m, nil, true

	case finderResultMsg:
		m.ActiveOverlay = overlayNone
		if msg.mode == finderModeFile {
			return m, openFilesInEditorCmd(msg.results), true
		}

		m.Chat.TextArea.Focus()
		originalContent := m.Chat.TextArea.Value()

		var commandToRun string
		if strings.HasPrefix(msg.result, "/") {
			commandToRun = msg.result
		} else if after, ok := strings.CutPrefix(msg.result, "model: "); ok {
			commandToRun = "/model " + after
		} else {
			commandToRun = "/model " + msg.result
		}
		m.Chat.TextArea.SetValue(commandToRun)
		m.Chat.TextArea.CursorEnd()
		m.Chat.PreserveInputOnSubmit = true
		model, cmd := m.handleSubmit()

		if newModel, ok := model.(Model); ok {
			newModel.Chat.TextArea.SetValue(originalContent)
			newModel.Chat.TextArea.CursorEnd()
			newModel = newModel.updateLayout()
			return newModel, cmd, true
		}
		return model, cmd, true

	case clearStatusBarMsg:
		m.StatusBarMessage = ""
		return m, nil, true

	case ctrlCTimeoutMsg:
		m.Chat.CtrlCPressed = false
		return m, nil, true

	case initialContextLoadedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Error loading initial context:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			if m.ActiveOverlay == overlayNone {
				m.Chat.Viewport.SetContent(m.renderConversation())
				m.Chat.Viewport.GotoBottom()
			}
			return m, nil, true
		}

		if m.Chat.AutoSubmitPending && m.Chat.TextArea.Value() != "" {
			m.Chat.AutoSubmitPending = false
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}

		m.UpdateTokenCount()
		return m, nil, true

	case termFinishedMsg:
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
				Type:     resType,
				Content:  content,
				Metadata: map[string]any{"canAISee": true, "isShell": true},
			})
		}
		m.State = stateIdle
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.Chat.TextArea.Reset()
		m.UpdateTokenCount()
		return m, textarea.Blink, true

	case errorMsg:
		m.Chat.IsStreaming = false

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		messages := m.Session.GetMessages()
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if lastMsg.Type == types.AIMessage && strings.TrimSpace(lastMsg.Content) != "" {
				m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			} else {
				lastMsg.Content = errorContent
				lastMsg.Type = types.CommandErrorResultMessage
				m.Session.ReplaceLastMessage(lastMsg)
			}
		}
		m.Chat.LastInteractionFailed = true

		m.State = stateIdle
		wasAtBottom := m.Chat.Viewport.AtBottom()
		m.Chat.Viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.Chat.Viewport.GotoBottom()
		}
		m.Chat.StreamSub = nil
		m.Session.CancelGeneration()
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.Focus()
		return m, saveConversationCmd(m.Session), true

	case tea.WindowSizeMsg:
		m.Height = msg.Height
		m.Width = msg.Width
		m.Chat.TextArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalFrameSize())
		m.Chat.Viewport.Width = msg.Width
		m = m.updateLayout()
		m.Chat.TextArea.CursorEnd()
		m.ClearCache()

		m.Chat.CtrlCPressed = false

		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle(m.Session.GetConfig().UI.MarkdownTheme),
			glamour.WithWordWrap(m.Chat.Viewport.Width),
		)
		if err == nil {
			m.GlamourRenderer = renderer
			m.Chat.Viewport.SetContent(m.renderConversation())
		}
		return m, nil, false
	}
	return m, nil, false
}
