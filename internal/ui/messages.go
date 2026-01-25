package ui

import (
	"fmt"
	"slices"
	"os"
	"path/filepath"
	"strings"

	"coder/internal/types"
	"coder/internal/utils"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

func (m Model) handleMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tokenizerInitializedMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Tokenizer init failed: %v", msg.err)
			// We can continue without a tokenizer, it will just use estimates.
		}
		m.State = stateIdle
		return m, loadInitialContextCmd(m.Session), true

	case modelsFetchedMsg:
		m.IsFetchingModels = false
		if msg.err != nil {
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: fmt.Sprintf("Failed to fetch models: %v", msg.err),
			})
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
			return m, nil, true
		}

		cfg := m.Session.GetConfig()
		cfg.AvailableModels = msg.models

		if len(msg.models) == 0 {
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: "Warning: Server returned no available models.",
			})
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
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
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
		}
		return m, nil, true
	case startGenerationMsg:
		if m.State != stateGenPending {
			return m, nil, true // Debounce was cancelled
		}

		event := m.Session.StartGeneration()

		switch event.Type {
		case types.GenerationStarted:
			model, cmd := m.startGeneration(event)
			return model, cmd, true
		}

		switch event.Type {
		case types.MessagesUpdated:
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
			m.State = stateIdle
			m.TextArea.Focus()
			return m, textarea.Blink, true
		}
		return m, nil, true

	case spinner.TickMsg:
		// Tick the spinner during all generation phases.
		switch m.State {
		case stateInitializing, stateGenPending, stateThinking, stateGenerating, stateCancelling:
			// Continue to spinner update logic
		default:
			return m, nil, true
		}

		var spinnerCmd tea.Cmd
		m.Spinner, spinnerCmd = m.Spinner.Update(msg)

		// If we are in the "thinking" state, the spinner is in the viewport.
		// We need to update the viewport's content to reflect the spinner's animation.
		switch m.State {
		case stateThinking, stateGenPending:
			wasAtBottom := m.Viewport.AtBottom()
			m.Viewport.SetContent(m.renderConversation())
			if wasAtBottom {
				m.Viewport.GotoBottom()
			}
		}
		return m, spinnerCmd, true

	case streamResultMsg:
		messages := m.Session.GetMessages()
		lastMsg := messages[len(messages)-1]
		switch m.State {
		case stateThinking:
			m.State = stateGenerating
			lastMsg.Content += string(msg)
			m.Session.ReplaceLastMessage(lastMsg)
			return m, tea.Batch(listenForStream(m.StreamSub), renderTick()), true
		}
		lastMsg.Content += string(msg)
		m.Session.ReplaceLastMessage(lastMsg)
		return m, listenForStream(m.StreamSub), true

	case streamFinishedMsg:
		if !m.IsStreaming {
			return m, nil, true
		}

		m.IsStreaming = false
		messages := m.Session.GetMessages()

		switch m.State {
		case stateCancelling:
			// This was a cancellation.
			lastMsg := messages[len(messages)-1]
			lastMsg.Content = "Generation cancelled."
			lastMsg.Type = types.CommandResultMessage // Re-use style for notification
			m.Session.ReplaceLastMessage(lastMsg)
			m.LastInteractionFailed = true
		}

		wasAtBottom := m.Viewport.AtBottom()
		m.Viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.Viewport.GotoBottom()
		}

		m.State = stateIdle
		m.StreamSub = nil
		m.Session.CancelGeneration()
		m.TextArea.Reset()
		m = m.updateLayout()
		m.TextArea.Focus()

		if m.LastInteractionFailed {
			return m, nil, true // Don't count tokens on failure/cancellation
		}

		prompt := m.Session.GetPrompt()
		m.IsCountingTokens = true

		return m, tea.Batch(countTokensCmd(prompt), saveConversationCmd(m.Session)), true

	case editorFinishedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Editor Error:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
			m.EditingMessageIndex = -1 // Also reset here
			return m, nil, true
		}

		if m.EditingMessageIndex != -1 {
			// This block handles the return from editing a previous message in the history.
			// It updates the message in place and does not trigger a new generation.
			if msg.content != msg.originalContent {
				if err := m.Session.EditMessage(m.EditingMessageIndex, msg.content); err != nil {
					// This should ideally not happen if the logic for selecting an editable message is correct.
					errorContent := fmt.Sprintf("\n**Editor Error:**\n```\nFailed to apply edit: %v\n```\n", err)
					m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
				}
			}

			var cmd tea.Cmd
			if m.IsStreaming {
				messages := m.Session.GetMessages()
				if len(messages) > 0 && messages[len(messages)-1].Type == types.AIMessage && messages[len(messages)-1].Content == "" {
					m.State = stateThinking
				} else {
					m.State = stateGenerating
				}
				cmd = m.Spinner.Tick
			} else {
				m.State = stateIdle
				m.TextArea.Focus()
				cmd = textarea.Blink
			}

			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()

			m.EditingMessageIndex = -1 // Reset on success or failure
			m.IsCountingTokens = true
			return m, tea.Batch(cmd, countTokensCmd(m.Session.GetPrompt())), true
		}

		// This is for Ctrl+E on the text area. If content changed, submit.
		if msg.content != msg.originalContent {
			m.TextArea.SetValue(msg.content)
			m.TextArea.CursorEnd()
			model, cmd := m.handleSubmit()
			return model, cmd, true
		}

		// Content is unchanged, just update textarea and focus.
		m.TextArea.SetValue(msg.originalContent)
		m.TextArea.Focus()
		return m, textarea.Blink, true

	case renderTickMsg:
		if m.State != stateGenerating || !m.IsStreaming {
			return m, nil, true
		}

		messages := m.Session.GetMessages()
		lastMsg := messages[len(messages)-1]
		if lastMsg.Content != m.LastRenderedAIPart {
			wasAtBottom := m.Viewport.AtBottom()
			m.Viewport.SetContent(m.renderConversation())
			if wasAtBottom {
				m.Viewport.GotoBottom()
			}
			m.LastRenderedAIPart = lastMsg.Content
		}

		return m, renderTick(), true

	case tokenCountResultMsg:
		m.TokenCount = int(msg)
		m.IsCountingTokens = false
		return m, nil, true

	case historyListResultMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading history: %v", msg.err)
			m.State = stateIdle
			m.TextArea.Focus()
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}
		m.HistoryItems = msg.items

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
		m.HistoryCussorPos = initialCursorPos

		m.Viewport.SetContent(m.historyView())
		// Center the selected item in the viewport
		const headerHeight = 2 // "Select a conversation...\n\n"
		targetOffset := (initialCursorPos + headerHeight) - (m.Viewport.Height / 2)
		m.Viewport.SetYOffset(targetOffset)
		return m, nil, true

	case conversationLoadedMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading conversation: %v", msg.err)
			m.State = stateIdle
			m.TextArea.Focus()
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		welcome := types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage}
		dirInfo := types.Message{Type: types.DirectoryMessage, Content: utils.GetDirInfoContent()}
		m.Session.PrependMessages(welcome, dirInfo)

		m.State = stateIdle
		m.LastInteractionFailed = false
		m.LastRenderedAIPart = ""
		m.TextArea.Reset()
		m.TextArea.SetHeight(1)
		m.TextArea.Focus()
		m.Viewport.SetContent(m.renderConversation())
		m.Viewport.GotoBottom()
		m.IsCountingTokens = true
		return m, tea.Batch(countTokensCmd(m.Session.GetPrompt()), textarea.Blink), true

	case titleGeneratedMsg:
		m.AnimatingTitle = true
		m.FullGeneratedTitle = msg.title
		m.DisplayedTitle = ""
		return m, animateTitleTick(), true

	case pasteResultMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Paste error: %v", msg.err)
			return m, clearStatusBarCmd(), true
		}

		if msg.isImage {
			m.Session.AddMessages(types.Message{Type: types.ImageMessage, Content: msg.content})
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
		} else {
			m.TextArea.InsertString(msg.content)
		}
		return m, nil, false

	case animateTitleTickMsg:
		if !m.AnimatingTitle {
			return m, nil, true
		}

		if len(m.DisplayedTitle) < len(m.FullGeneratedTitle) {
			// Use rune-safe slicing to handle multi-byte characters
			m.DisplayedTitle = string([]rune(m.FullGeneratedTitle)[:len([]rune(m.DisplayedTitle))+1])
			return m, animateTitleTick(), true
		}

		m.AnimatingTitle = false
		return m, nil, true

	case finderResultMsg:
		m.State = stateIdle
		m.TextArea.Focus()
		originalContent := m.TextArea.Value()

		var commandToRun string
		parts := strings.SplitN(msg.result, ": ", 2)
		if len(parts) == 2 {
			commandToRun = fmt.Sprintf(":%s %s", parts[0], parts[1])
		} else {
			commandToRun = ":" + msg.result
		}
		m.TextArea.SetValue(commandToRun)
		m.TextArea.CursorEnd()
		m.PreserveInputOnSubmit = true
		model, cmd := m.handleSubmit()

		if newModel, ok := model.(Model); ok {
			newModel.TextArea.SetValue(originalContent)
			newModel.TextArea.CursorEnd()
			return newModel, cmd, true
		}
		return model, cmd, true

	case searchResultMsg:
		m.State = stateIdle
		m.TextArea.Focus()

		content, offsets := m.renderConversationWithOffsets()
		m.messageLineOffsets = offsets
		m.Viewport.SetContent(content)

		if line, ok := m.messageLineOffsets[msg.item.MsgIndex]; ok {
			// Determine if there's a border offset based on message type
			borderOffset := 0
			messageType := m.Session.GetMessages()[msg.item.MsgIndex].Type
			if messageType == types.UserMessage { // Add other types here if they get borders
				borderOffset = 1
			}

			// Calculate the absolute line number of the found text
			absoluteLine := line + borderOffset + msg.item.LineNum

			// Center the found line in the viewport
			offset := absoluteLine - (m.Viewport.Height / 2)
			if offset < 0 {
				offset = 0
			}
			m.Viewport.SetYOffset(offset)
			m.StatusBarMessage = fmt.Sprintf("Jumped to message %d, line %d.", msg.item.MsgIndex, msg.item.LineNum+1)
		} else {
			m.StatusBarMessage = fmt.Sprintf("Found match, but couldn't jump. See message %d.", msg.item.MsgIndex)
		}

		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true

	case treeReadyMsg:
		m.Tree.root = msg.root
		m.Tree.expandSelectedNodes()
		m.Tree.buildVisibleNodes()
		return m, nil, true

	case treeSelectionResultMsg:
		m.State = stateIdle
		m.TextArea.Focus()

		repoRoot := utils.GetProjectRoot()
		cwd, err := os.Getwd()
		if err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error getting current directory: %v", err)
			return m, clearStatusBarCmd(), true
		}

		cfg := m.Session.GetConfig()
		cfg.Context.Files = []string{}
		cfg.Context.Dirs = []string{}

		for _, p := range msg.selectedPaths {
			absPath := filepath.Join(repoRoot, p)
			info, err := os.Stat(absPath)
			if err != nil {
				continue // ignore paths that don't exist
			}
			relToCwd, err := filepath.Rel(cwd, absPath)
			if err != nil {
				relToCwd = absPath // fallback to absolute path
			}
			if info.IsDir() {
				cfg.Context.Dirs = append(cfg.Context.Dirs, relToCwd)
			} else {
				cfg.Context.Files = append(cfg.Context.Files, relToCwd)
			}
		}

		if err := m.Session.LoadContext(); err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading context: %v", err)
			return m, clearStatusBarCmd(), true
		}

		event := m.Session.HandleInput(":list")
		model, cmd := m.handleEvent(event)
		if newModel, ok := model.(Model); ok {
			return newModel, cmd, true
		}
		return model, cmd, true

	case clearStatusBarMsg:
		m.StatusBarMessage = ""
		return m, nil, true

	case ctrlCTimeoutMsg:
		m.CtrlCPressed = false
		return m, nil, true

	case initialContextLoadedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Error loading initial context:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			m.Viewport.SetContent(m.renderConversation())
			m.Viewport.GotoBottom()
			return m, nil, true
		}

		// Now that context is loaded, count the tokens.
		m.IsCountingTokens = true
		return m, countTokensCmd(m.Session.GetPrompt()), true

	case errorMsg:
		m.IsStreaming = false

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		messages := m.Session.GetMessages()
		lastMsg := messages[len(messages)-1]
		lastMsg.Content = errorContent
		m.Session.ReplaceLastMessage(lastMsg)
		m.LastInteractionFailed = true

		wasAtBottom := m.Viewport.AtBottom()
		m.Viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.Viewport.GotoBottom()
		}
		m.State = stateIdle
		m.StreamSub = nil
		m.Session.CancelGeneration()
		m.TextArea.Reset()
		m.TextArea.Focus()
		return m, nil, true

	case tea.WindowSizeMsg:
		m.Height = msg.Height
		m.Width = msg.Width
		m.TextArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalFrameSize())
		m.Viewport.Width = msg.Width

		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.Viewport.Width),
		)
		if err == nil {
			m.GlamourRenderer = renderer
			m.Viewport.SetContent(m.renderConversation())
		}
		return m, nil, false
	}
	return m, nil, false
}
