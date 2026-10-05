package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if newModel, cmd, handled := m.handleRenderMessage(msg); handled {
		return newModel, cmd, true
	}
	if newModel, cmd, handled := m.handleProcessMessage(msg); handled {
		return newModel, cmd, true
	}

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
		targetModel := cfg.Coder.ModelCode
		modelTypeLabel := "chat"
		if m.Session.Capabilities().Has(engine.CapToolLoop) {
			targetModel = cfg.Agent.ModelCode
			modelTypeLabel = "agent"
		}
		if !slices.Contains(msg.models, targetModel) {
			errorStrings = append(errorStrings, fmt.Sprintf("Configured %s model '%s' is not in the available list.", modelTypeLabel, targetModel))
			hasError = true
		}
		if !slices.Contains(msg.models, cfg.Title.ModelCode) {
			errorStrings = append(errorStrings, fmt.Sprintf("Configured title model '%s' is not in the available list.", cfg.Title.ModelCode))
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
		switch m.State {
		case stateAsking, stateThinking:
			m.Chat.Viewport.SetContent(m.renderConversation())
			if m.Chat.AutoScroll {
				m.Chat.Viewport.GotoBottom()
			}
		}
		return m, spinnerCmd, true

	case sessionEventMsg:
		targetSess := m.getSessionByID(msg.sessID)
		if targetSess == nil {
			return m, nil, true
		}

		isActive := m.Session != nil && targetSess.GetID() == m.Session.GetID()
		if isActive && (!m.Chat.IsStreaming || m.Chat.EventSub != msg.sub) {
			return m, listenForEvents(msg.sessID, msg.sub), true
		}

		if msg.event.Kind == types.EventError {
			m.Chat.IsStreaming = false
			m.Chat.IsAIRendering = false
			m.Chat.PendingAIRender = false
			m.Chat.LastInteractionFailed = true
			m.State = stateIdle
			m.StatusText = ""
			if m.ActiveOverlay == overlayConfirm {
				m.ActiveOverlay = overlayNone
			}
			m.ConfirmRequest = nil
			if isActive {
				m.Chat.Viewport.SetContent(m.renderConversation())
				if m.Chat.AutoScroll {
					m.Chat.Viewport.GotoBottom()
				}
				m.Chat.EventSub = nil
				m.Chat.TextArea.Reset()
				m.Chat.TextArea.Focus()
			}
			return m, saveConversationCmd(targetSess), true
		}

		if msg.event.Kind == types.EventToolConfirm && isActive {
			m.ConfirmRequest = msg.event.Confirm
			m.ActiveOverlay = overlayConfirm
			m.Chat.Viewport.SetContent(m.renderConversation())
			if m.Chat.AutoScroll {
				m.Chat.Viewport.GotoBottom()
			}
			return m, listenForEvents(msg.sessID, msg.sub), true
		}

		if msg.event.Kind == types.EventThinking && isActive {
			if m.State != stateGenerating {
				m.State = stateThinking
			}
			if msg.event.Content != "" {
				m.StatusText = msg.event.Content
			}
		}

		if msg.event.Kind == types.EventToolCall && isActive {
			m.State = stateThinking
			if msg.event.ToolName != "" {
				m.StatusText = fmt.Sprintf("Running %s", msg.event.ToolName)
			} else {
				m.StatusText = "Running tool"
			}
		}

		if msg.event.Kind == types.EventToolResult && isActive {
			m.State = stateThinking
			m.StatusText = "Processing"
		}

		var renderCmd tea.Cmd
		if msg.event.Kind == types.EventChunk && msg.event.Content != "" {
			if isActive {
				m.State = stateGenerating
				m.Chat.StateStartTime = time.Now()
				m.StatusText = "Generating"
			}
			if isActive {
				if !m.Chat.IsAIRendering {
					m.Chat.IsAIRendering = true
					m.Chat.PendingAIRender = false
					viewportWidth := max(10, m.Chat.Viewport.Width)
					msgs := targetSess.GetMessages()
					aiIdx := len(msgs) - 1
					latestContent := ""
					if aiIdx >= 0 {
						latestContent = msgs[aiIdx].Content
					}
					renderCmd = renderAIMessageCmd(msg.sessID, aiIdx, latestContent, viewportWidth)
				} else {
					m.Chat.PendingAIRender = true
				}
			}
		}

		if isActive && (msg.event.Kind == types.EventToolCall || msg.event.Kind == types.EventToolResult || msg.event.Kind == types.EventThinking) {
			m.Chat.Viewport.SetContent(m.renderConversation())
			if m.Chat.AutoScroll {
				m.Chat.Viewport.GotoBottom()
			}
		}

		return m, tea.Batch(listenForEvents(msg.sessID, msg.sub), renderCmd), true

	case sessionFinishedMsg:
		targetSess := m.getSessionByID(msg.sessID)
		if targetSess == nil {
			return m, nil, true
		}
		targetSess.Cancel()

		messages := targetSess.GetMessages()
		if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
			targetSess.DeleteMessages([]int{len(messages) - 1})
		}

		isActive := m.Session != nil && targetSess.GetID() == m.Session.GetID()
		if !isActive || !m.Chat.IsStreaming {
			return m, saveConversationCmd(targetSess), true
		}

		m.Chat.IsStreaming = false
		m.State = stateIdle
		if m.ActiveOverlay == overlayConfirm {
			m.ActiveOverlay = overlayNone
		}
		m.ConfirmRequest = nil
		if m.ActiveOverlay == overlayNone {
			m.Chat.TextArea.Focus()
		}

		m.Chat.EventSub = nil
		m.Chat.TextArea.Reset()
		m.StatusText = ""
		m = m.updateLayout()

		var cmds []tea.Cmd
		cmds = append(cmds, saveConversationCmd(targetSess), m.Chat.Spinner.Tick, m.renderUncachedCmd())
		if !m.Chat.LastInteractionFailed {
			cmds = append(cmds, m.updateTokenCountCmd())
		}

		newModel, renderCmd := m.finalizeAIMessageRender(msg.sessID)
		if renderCmd != nil {
			cmds = append(cmds, renderCmd)
		}
		return newModel, tea.Batch(cmds...), true

	case historyListResultMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading history: %v", msg.err)
			if m.ActiveOverlay == overlaySelector {
				m.ActiveOverlay = overlayNone
			}
			if m.State == stateIdle && m.ActiveOverlay == overlayNone {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		var selectorItems []SelectorItem
		for _, item := range msg.items {
			dateStr := ""
			displayTime := item.ModifiedAt
			if displayTime.IsZero() {
				displayTime = item.CreatedAt
			}
			if !displayTime.IsZero() {
				dateStr = fmt.Sprintf(" (%s)", displayTime.Format("2006-01-02 15:04"))
			}
			selectorItems = append(selectorItems, SelectorItem{
				ID:          item.Filename,
				Title:       item.Title,
				Description: dateStr,
				Data:        item,
			})
		}

		if m.ActiveOverlay == overlaySelector && m.Selector.ActiveTab == 0 {
			m.Selector.SetItems(selectorItems)
			currentFilename := m.Session.GetHistoryFilename()
			if currentFilename != "" {
				for i, item := range selectorItems {
					if item.ID == currentFilename {
						m.Selector.Cursor = i
						break
					}
				}
			}
		}
		return m, nil, true

	case addFilesListResultMsg:
		if m.ActiveOverlay != overlaySelector || !m.Selector.IsLoading {
			return m, nil, true
		}

		m.Selector.IsLoading = false
		if len(msg.items) == 0 {
			m.ActiveOverlay = overlayNone
			m.StatusBarMessage = "No files or directories found."
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		var items []SelectorItem
		for _, p := range msg.items {
			items = append(items, SelectorItem{
				ID:    p,
				Title: p,
			})
		}
		m.Selector.SetItems(items)
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

		oldSess := m.Session

		m.ActiveOverlay = overlayNone
		m.Session = msg.sess
		m.ClearCache()
		m.addActiveSession(msg.sess)

		welcome := types.Message{Type: types.InitMessage, Content: welcomeMessage}
		dirInfo := types.Message{Type: types.DirectoryMessage, Content: project.DirInfo()}
		m.Session.PrependMessages(welcome, dirInfo)

		if m.Session.IsStreaming() {
			messages := m.Session.GetMessages()
			if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content != "" {
				m.State = stateGenerating
			} else {
				m.State = stateAsking
			}
			m.Chat.IsStreaming = true
			m.Chat.TextArea.Blur()
		} else {
			m.State = stateIdle
			m.Chat.IsStreaming = false
			m.Chat.IsAIRendering = false
			m.Chat.PendingAIRender = false
			m.Chat.TextArea.Focus()
		}

		m.Chat.LastInteractionFailed = false
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.SetHeight(1)
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		var cmds []tea.Cmd
		cmds = append(cmds, textarea.Blink, m.updateTokenCountCmd(), m.renderUncachedCmd())
		if oldSess != nil && oldSess.GetID() != msg.sess.GetID() {
			cmds = append(cmds, saveConversationCmd(oldSess))
		}
		return m, tea.Batch(cmds...), true

	case switchActiveSessionMsg:
		oldSess := m.Session
		m.ActiveOverlay = overlayNone
		m.Session = msg.sess
		m.ClearCache()
		if m.Session.IsStreaming() {
			messages := m.Session.GetMessages()
			if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content != "" {
				m.State = stateGenerating
			} else {
				m.State = stateAsking
			}
			m.Chat.IsStreaming = true
			m.Chat.TextArea.Blur()
		} else {
			m.State = stateIdle
			m.Chat.IsStreaming = false
			m.Chat.IsAIRendering = false
			m.Chat.PendingAIRender = false
			m.Chat.TextArea.Focus()
		}
		m.Chat.LastInteractionFailed = false
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.SetHeight(1)
		_ = m.Session.LoadContext()

		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		var cmds []tea.Cmd
		cmds = append(cmds, textarea.Blink, m.updateTokenCountCmd(), m.renderUncachedCmd())
		if oldSess != nil && oldSess.GetID() != msg.sess.GetID() {
			cmds = append(cmds, saveConversationCmd(oldSess))
		}
		return m, tea.Batch(cmds...), true

	case titleGeneratedMsg:
		m.Chat.AnimatingTitle = true
		m.Chat.FullGeneratedTitle = msg.title
		m.Chat.DisplayedTitle = ""
		return m, animateTitleTick(), true

	case PasteResultMsg:
		if msg.Err != nil {
			m.StatusBarMessage = fmt.Sprintf("Paste error: %v", msg.Err)
			return m, clearStatusBarCmd(), true
		}

		if msg.IsImage {
			m.Session.AddMessages(types.Message{Type: types.ImageMessage, Content: msg.Content})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			return m, m.updateTokenCountCmd(), true
		} else {
			m.Chat.TextArea.InsertString(msg.Content)
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

		return m, tea.Batch(m.updateTokenCountCmd(), m.renderUncachedCmd()), true

	case errorMsg:
		targetSess := m.getSessionByID(msg.sessID)
		if targetSess == nil {
			return m, nil, true
		}
		targetSess.Cancel()

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		messages := targetSess.GetMessages()
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if lastMsg.Type == types.AIMessage && strings.TrimSpace(lastMsg.Content) != "" {
				targetSess.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			} else {
				lastMsg.Content = errorContent
				lastMsg.Type = types.CommandErrorResultMessage
				targetSess.ReplaceLastMessage(lastMsg)
			}
		}

		if m.Session != nil && targetSess.GetID() == m.Session.GetID() {
			m.Chat.IsStreaming = false
			m.Chat.IsAIRendering = false
			m.Chat.PendingAIRender = false
			m.Chat.LastInteractionFailed = true
			m.State = stateIdle
			m.Chat.Viewport.SetContent(m.renderConversation())
			if m.Chat.AutoScroll {
				m.Chat.Viewport.GotoBottom()
			}
			m.Chat.EventSub = nil
			m.Chat.TextArea.Reset()
			m.Chat.TextArea.Focus()
		}
		return m, saveConversationCmd(targetSess), true

	case tokenCountResultMsg:
		if m.Session != nil && m.Session.GetID() == msg.sessID {
			m.TokenCount = msg.count
		}
		return m, nil, true

	case tea.WindowSizeMsg:
		widthChanged := m.Width != msg.Width
		m.Height = msg.Height
		m.Width = msg.Width
		m.Chat.TextArea.SetWidth(msg.Width - TextAreaContainerStyle.GetHorizontalFrameSize())
		m.Chat.Viewport.Width = msg.Width
		m = m.updateLayout()
		m.Chat.TextArea.CursorEnd()

		m.Chat.CtrlCPressed = false

		if widthChanged {
			m.ClearCache()
			renderer, err := markdown.NewRenderer(m.Chat.Viewport.Width)
			if err == nil {
				m.GlamourRenderer = renderer
			}
			m.Chat.Viewport.SetContent(m.renderConversation())
			if m.Chat.AutoScroll || m.Chat.Viewport.PastBottom() {
				m.Chat.Viewport.GotoBottom()
			}
			return m, m.renderUncachedCmd(), false
		}
		if m.Chat.AutoScroll || m.Chat.Viewport.PastBottom() {
			m.Chat.Viewport.GotoBottom()
		}
		return m, nil, false
	}
	return m, nil, false
}
