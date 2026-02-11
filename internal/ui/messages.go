package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"

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
		m.Chat.IsFetchingModels = false
		if msg.err != nil {
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: fmt.Sprintf("Failed to fetch models: %v", msg.err),
			})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			return m, nil, true
		}

		cfg := m.Session.GetConfig()
		cfg.AvailableModels = msg.models

		if len(msg.models) == 0 {
			m.Session.AddMessages(types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: "Warning: Server returned no available models.",
			})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
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
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
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
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			m.State = stateIdle
			m.Chat.TextArea.Focus()
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
		m.Chat.Spinner, spinnerCmd = m.Chat.Spinner.Update(msg)

		// If we are in the "thinking" state, the spinner is in the viewport.
		// We need to update the viewport's content to reflect the spinner's animation.
		switch m.State {
		case stateThinking, stateGenPending:
			wasAtBottom := m.Chat.Viewport.AtBottom()
			m.Chat.Viewport.SetContent(m.renderConversation())
			if wasAtBottom {
				m.Chat.Viewport.GotoBottom()
			}
		}
		return m, spinnerCmd, true

	case streamResultMsg:
		if m.State == stateThinking {
			m.State = stateGenerating
		}
		delay := m.Session.GetConfig().Generation.StreamDelay
		m.Chat.StreamBuffer += string(msg)
		if !m.Chat.IsStreamAnime {
			m.Chat.IsStreamAnime = true
			return m, tea.Batch(listenForStream(m.Chat.StreamSub), streamAnimeCmd(delay)), true
		}
		return m, listenForStream(m.Chat.StreamSub), true

	case streamAnimeMsg:
		if m.Chat.StreamBuffer == "" {
			if m.Chat.StreamDone {
				return m, func() tea.Msg { return streamFinishedMsg{} }, true
			}
			m.Chat.IsStreamAnime = false
			return m, nil, true
		}

		// Adaptive anime: take more characters if the buffer is getting large
		take := 1
		bufLen := len(m.Chat.StreamBuffer)
		if bufLen > 300 {
			take = bufLen / 10
		} else if bufLen > 50 {
			take = 4
		} else if bufLen > 10 {
			take = 2
		}

		if take > len(m.Chat.StreamBuffer) {
			take = len(m.Chat.StreamBuffer)
		}

		chunk := m.Chat.StreamBuffer[:take]
		m.Chat.StreamBuffer = m.Chat.StreamBuffer[take:]

		messages := m.Session.GetMessages()
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			lastMsg.Content += chunk
			m.Session.ReplaceLastMessage(lastMsg)

			if lastMsg.Content != m.Chat.LastRenderedAIPart {
				wasAtBottom := m.Chat.Viewport.AtBottom()
				m.Chat.Viewport.SetContent(m.renderConversation())
				if wasAtBottom {
					m.Chat.Viewport.GotoBottom()
				}
				m.Chat.LastRenderedAIPart = lastMsg.Content
			}
		}

		delay := m.Session.GetConfig().Generation.StreamDelay
		return m, streamAnimeCmd(delay), true

	case streamFinishedMsg:
		if !m.Chat.IsStreaming || m.Chat.StreamBuffer != "" {
			m.Chat.StreamDone = true
			return m, nil, true
		}

		m.Chat.IsStreaming = false
		m.Chat.IsStreamAnime = false
		m.Chat.StreamBuffer = ""
		m.Chat.StreamDone = false

		messages := m.Session.GetMessages()

		switch m.State {
		case stateCancelling:
			// This was a cancellation.
			lastMsg := messages[len(messages)-1]
			lastMsg.Content = "Generation cancelled."
			lastMsg.Type = types.CommandResultMessage // Re-use style for notification
			m.Session.ReplaceLastMessage(lastMsg)
			m.Chat.LastInteractionFailed = true
		}

		wasAtBottom := m.Chat.Viewport.AtBottom()
		m.Chat.Viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.Chat.Viewport.GotoBottom()
		}

		m.State = stateIdle
		m.Chat.StreamSub = nil
		m.Session.CancelGeneration()
		m.Chat.TextArea.Reset()
		m = m.updateLayout()
		m.Chat.TextArea.Focus()

		if m.Chat.LastInteractionFailed {
			return m, nil, true // Don't count tokens on failure/cancellation
		}

		prompt := m.Session.GetPrompt()
		m.IsCountingTokens = true

		return m, tea.Batch(countTokensCmd(prompt), saveConversationCmd(m.Session), m.Chat.Spinner.Tick), true

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
					m.State = stateThinking
				} else {
					m.State = stateGenerating
				}
				cmd = m.Chat.Spinner.Tick
			} else {
				m.State = stateIdle
				m.Chat.TextArea.Focus()
				cmd = textarea.Blink
			}

			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()

			m.Chat.EditingMessageIndex = -1 // Reset on success or failure
			m.IsCountingTokens = true
			return m, tea.Batch(cmd, countTokensCmd(m.Session.GetPrompt())), true
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

	case tokenCountResultMsg:
		m.TokenCount = int(msg)
		m.IsCountingTokens = false
		return m, nil, true

	case historyListResultMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading history: %v", msg.err)
			m.State = stateIdle
			m.Chat.TextArea.Focus()
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

		m.Chat.Viewport.SetContent(m.historyListView())
		m.centerHistoryViewport()
		return m, nil, true

	case conversationLoadedMsg:
		if msg.err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error loading conversation: %v", msg.err)
			m.State = stateIdle
			m.Chat.TextArea.Focus()
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		welcome := types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage}
		dirInfo := types.Message{Type: types.DirectoryMessage, Content: utils.GetDirInfoContent()}
		m.Session.PrependMessages(welcome, dirInfo)

		m.State = stateIdle
		m.Chat.LastInteractionFailed = false
		m.Chat.LastRenderedAIPart = ""
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.SetHeight(1)
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.IsCountingTokens = true
		return m, tea.Batch(countTokensCmd(m.Session.GetPrompt()), textarea.Blink), true

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
		m.State = stateIdle
		m.Chat.TextArea.Focus()
		originalContent := m.Chat.TextArea.Value()

		var commandToRun string
		parts := strings.SplitN(msg.result, ": ", 2)
		if len(parts) == 2 {
			commandToRun = fmt.Sprintf(":%s %s", parts[0], parts[1])
		} else {
			commandToRun = ":" + msg.result
		}
		m.Chat.TextArea.SetValue(commandToRun)
		m.Chat.TextArea.CursorEnd()
		m.Chat.PreserveInputOnSubmit = true
		model, cmd := m.handleSubmit()

		if newModel, ok := model.(Model); ok {
			newModel.Chat.TextArea.SetValue(originalContent)
			newModel.Chat.TextArea.CursorEnd()
			return newModel, cmd, true
		}
		return model, cmd, true

	case searchResultMsg:
		m.State = stateIdle
		m.Chat.TextArea.Focus()

		m.Chat.SearchQuery = msg.query
		m.Chat.SearchFocusMsgIndex = msg.item.MsgIndex
		m.Chat.SearchFocusLineNum = msg.item.LineNum

		content, offsets := m.renderConversationWithOffsets()
		m.Chat.MessageLineOffsets = offsets
		m.Chat.Viewport.SetContent(content)

		if line, ok := m.Chat.MessageLineOffsets[msg.item.MsgIndex]; ok {
			// Determine if there's a border offset based on message type
			borderOffset := 0
			messageType := m.Session.GetMessages()[msg.item.MsgIndex].Type
			if messageType == types.UserMessage { // Add other types here if they get borders
				borderOffset = 1
			}

			// Calculate the absolute line number of the found text
			absoluteLine := line + borderOffset + msg.item.LineNum

			// Center the found line in the viewport
			offset := absoluteLine - (m.Chat.Viewport.Height / 2)
			if offset < 0 {
				offset = 0
			}
			m.Chat.Viewport.SetYOffset(offset)
			m.StatusBarMessage = fmt.Sprintf("Jumped to message %d, line %d.", msg.item.MsgIndex, msg.item.LineNum+1)
		} else {
			m.StatusBarMessage = fmt.Sprintf("Found match, but couldn't jump. See message %d.", msg.item.MsgIndex)
		}

		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true

	case jumpResultMsg:
		m.State = stateIdle
		m.Chat.TextArea.Focus()

		content, offsets := m.renderConversationWithOffsets()
		m.Chat.MessageLineOffsets = offsets
		m.Chat.Viewport.SetContent(content)

		if line, ok := m.Chat.MessageLineOffsets[msg.msgIndex]; ok {
			// Determine if there's a border offset based on message type
			borderOffset := 0
			messageType := m.Session.GetMessages()[msg.msgIndex].Type
			if messageType == types.UserMessage {
				borderOffset = 1
			}

			absoluteLine := line + borderOffset

			// Center the message in the viewport
			offset := absoluteLine - (m.Chat.Viewport.Height / 2)
			if offset < 0 {
				offset = 0
			}
			m.Chat.Viewport.SetYOffset(offset)
			m.StatusBarMessage = fmt.Sprintf("Jumped to message %d.", msg.msgIndex)
		} else {
			m.StatusBarMessage = fmt.Sprintf("Could not find line offset for message %d.", msg.msgIndex)
		}

		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true

	case treeReadyMsg:
		m.Tree.root = msg.root
		m.Tree.expandSelectedNodes()
		m.Tree.buildVisibleNodes()
		return m, nil, true

	case treeSelectionResultMsg:
		m.State = stateIdle
		m.Chat.TextArea.Focus()

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
		m.Chat.CtrlCPressed = false
		return m, nil, true

	case initialContextLoadedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Error loading initial context:**\n```\n%v\n```\n", msg.err)
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: errorContent})
			m.Chat.Viewport.SetContent(m.renderConversation())
			m.Chat.Viewport.GotoBottom()
			return m, nil, true
		}

		// Now that context is loaded, count the tokens.
		m.IsCountingTokens = true
		return m, countTokensCmd(m.Session.GetPrompt()), true

	case errorMsg:
		m.Chat.IsStreaming = false

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		messages := m.Session.GetMessages()
		lastMsg := messages[len(messages)-1]
		lastMsg.Content = errorContent
		m.Session.ReplaceLastMessage(lastMsg)
		m.Chat.LastInteractionFailed = true

		wasAtBottom := m.Chat.Viewport.AtBottom()
		m.Chat.Viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.Chat.Viewport.GotoBottom()
		}
		m.State = stateIdle
		m.Chat.StreamSub = nil
		m.Session.CancelGeneration()
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.Focus()
		return m, nil, true

	case tea.WindowSizeMsg:
		m.Height = msg.Height
		m.Width = msg.Width
		m.Chat.TextArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalFrameSize())
		m.Chat.Viewport.Width = msg.Width
		m = m.updateLayout()
		m.Chat.TextArea.CursorEnd()

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
