package coagentui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/types"
	coderui "github.com/sokinpui/coder/internal/ui/coder"
	"github.com/sokinpui/coder/internal/ui/core"
	"github.com/sokinpui/coder/internal/ui/core/markdown"
)

type editorFinishedMsg struct {
	content string
	err     error
}

type ctrlCTimeoutMsg struct{}
type historyLoadedMsg struct{ filename string }
type historyListResultMsg struct {
	items []history.ConversationInfo
	err   error
}
type clearStatusBarMsg struct{}
type titleGeneratedMsg struct{ sessID, title string }
type animateTitleTickMsg struct{}
type aiRenderedMsg struct {
	sessID  string
	msgIdx  int
	content string
	lines   []string
	width   int
}
type precomputeToolMsg struct {
	idx int
	msg types.Message
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showSelector {
			return m.handleSelectorUpdate(msg)
		}
		return m.handleKey(msg)
	case initPromptMsg:
		return m.startPrompt(string(msg))
	case streamChunkMsg:
		return m.handleStreamChunk(msg.sessID, msg.chunk)
	case precomputeToolMsg:
		if m.session != nil {
			m.viewport.PrecomputeToolExpanded(msg.idx, msg.msg)
		}
		return m, nil
	case aiRenderedMsg:
		if m.session == nil || m.session.ID != msg.sessID {
			return m, nil
		}
		m.isAIRendering = false
		messages := m.session.Messages
		if msg.msgIdx < 0 || msg.msgIdx >= len(messages) {
			return m, nil
		}
		if messages[msg.msgIdx].Type != types.AIMessage {
			return m, nil
		}

		m.viewport.RenderCache[msg.msgIdx] = markdown.CachedRender{
			Lines:   msg.lines,
			Content: msg.content,
			Width:   msg.width,
		}

		m = m.updateViewportContent()
		if m.pendingAIRender || messages[msg.msgIdx].Content != msg.content || msg.width != m.viewport.Width {
			newModel, cmd := m.renderLastAIMessage(msg.sessID)
			return newModel, cmd
		}
		return m, nil
	case agentFinishedMsg:
		return m.handleAgentFinished(msg.sessID)
	case agentErrorMsg:
		return m.handleAgentError(msg.sessID, msg.err)
	case editorFinishedMsg:
		if msg.err != nil {
			m.editingMsgIdx = -1
			return m, nil
		}
		if m.editingMsgIdx != -1 {
			if msg.content != "" && m.session != nil && m.editingMsgIdx < len(m.session.Messages) {
				m.session.Messages[m.editingMsgIdx].Content = msg.content
				_ = m.session.SaveConversation()
				m.tokenCount = token.CountTokens(m.session.GetPrompt())
			}
			m.editingMsgIdx = -1
			m.input.Model.Focus()
			return m.updateViewportContent(), nil
		}
		if msg.content != "" {
			m.input.Model.Reset()
			return m.startPrompt(msg.content)
		}
		return m, nil
	case clearStatusBarMsg:
		m.statusBarMessage = ""
		return m, nil
	case titleGeneratedMsg:
		if m.session != nil && m.session.ID != msg.sessID {
			return m, nil
		}
		m.animatingTitle = true
		m.fullTitle = msg.title
		m.displayTitle = ""
		return m, animateTitleTick()
	case animateTitleTickMsg:
		if !m.animatingTitle {
			return m, nil
		}
		runes := []rune(m.fullTitle)
		dispRunes := []rune(m.displayTitle)
		if len(dispRunes) < len(runes) {
			m.displayTitle = string(runes[:len(dispRunes)+1])
			return m, animateTitleTick()
		}
		m.animatingTitle = false
		return m, nil
	case historyListResultMsg:
		if msg.err != nil || m.selector.ActiveTab != 0 {
			m.showSelector = false
			return m, nil
		}
		var items []coderui.SelectorItem
		for _, it := range msg.items {
			items = append(items, coderui.SelectorItem{
				ID:          it.Filename,
				Title:       it.Title,
				Description: it.CreatedAt.Format(" (2006-01-02 15:04)"),
				Data:        it,
			})
		}
		m.selector.SetItems(items)
		if m.session != nil && m.session.HistoryFilename != "" {
			for i, it := range items {
				if it.ID == m.session.HistoryFilename {
					m.selector.Cursor = i
					break
				}
			}
		}
		return m, nil
	case core.PasteResultMsg:
		if msg.Err != nil {
			return m, nil
		}
		if msg.IsImage {
			m.session.Messages = append(m.session.Messages, types.Message{
				Type:    types.ImageMessage,
				Content: msg.Content,
			})
			m.tokenCount = token.CountTokens(m.session.GetPrompt())
			return m.updateViewportContent(), nil
		}
		m.input.Model.InsertString(msg.Content)
		return m.updateLayout(), nil
	case ctrlCTimeoutMsg:
		m.ctrlCPressed = false
		return m.updateViewportContent(), nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.ctrlCPressed = false
		m = m.updateLayout()
		m.viewport.ClearCache()

		if m.session != nil && len(m.session.Messages) > 0 {
			lastIdx := len(m.session.Messages) - 1
			lastMsg := m.session.Messages[lastIdx]
			if (m.state == stateGenerating || m.state == stateThinking) && lastMsg.Type == types.AIMessage && lastMsg.Content != "" {
				m.isAIRendering = true
				m.pendingAIRender = false
				viewportWidth := max(10, m.viewport.Width)
				return m.updateViewportContent(), renderAIMessageCmd(m.session.ID, lastIdx, lastMsg.Content, viewportWidth)
			}
		}
		return m.updateViewportContent(), nil
	case spinner.TickMsg:
		if m.state != stateInput {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			m = m.updateViewportContent()
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m Model) startPrompt(prompt string) (tea.Model, tea.Cmd) {
	m.state = stateThinking
	m.stateStart = time.Now()
	m.statusText = "Thinking"
	m.isAIRendering = false
	m.pendingAIRender = false

	m.session.Messages = append(m.session.Messages, types.Message{
		Type:    types.UserMessage,
		Content: prompt,
	})
	m.tokenCount = token.CountTokens(m.session.GetPrompt())
	m = m.updateViewportContent()

	var cmds []tea.Cmd
	cmds = append(cmds, m.spinner.Tick)

	sessID := m.session.ID
	if !m.session.TitleGenerated {
		cmds = append(cmds, func() tea.Msg {
			title := m.session.GenerateTitle(context.Background(), prompt)
			_ = m.session.SaveConversation()
			return titleGeneratedMsg{sessID: sessID, title: title}
		})
	}

	m.chunkChan = make(chan coagent.AgentStreamChunk, 100)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	preparedMessages := m.session.PrepareMessages()
	go func() {
		m.session.Runtime.AgentLoop(ctx, m.session.Instruction, preparedMessages, m.chunkChan)
	}()

	cmds = append(cmds, waitForNextChunk(sessID, m.chunkChan))
	return m, tea.Batch(cmds...)
}

func (m Model) handleStreamChunk(sessID string, chunk coagent.AgentStreamChunk) (tea.Model, tea.Cmd) {
	targetSess := m.getSessionByID(sessID)
	if targetSess == nil {
		return m, nil
	}

	isActive := m.session != nil && targetSess.ID == m.session.ID

	if len(chunk.Messages) > 0 {
		targetSess.Messages = chunk.Messages
	}

	var precomputeCmd tea.Cmd
	if chunk.ToolCall != nil {
		if isActive {
			m.statusText = fmt.Sprintf("Running %s", chunk.ToolCall.Name)
		}
		targetSess.Messages = append(targetSess.Messages, types.Message{
			Type: types.ToolCallMessage,
			ToolCalls: []types.ToolCall{
				{
					ID:        chunk.ToolCall.CallID,
					Name:      chunk.ToolCall.Name,
					Arguments: chunk.ToolCall.Arguments,
				},
			},
		})
		callIdx := len(targetSess.Messages) - 1
		callMsg := targetSess.Messages[callIdx]
		precomputeCmd = func() tea.Msg { return precomputeToolMsg{idx: callIdx, msg: callMsg} }
	}

	if chunk.ToolResult != nil {
		targetSess.Messages = append(targetSess.Messages, types.Message{
			Type:       types.ToolResultMessage,
			Content:    chunk.ToolResult.Output,
			ToolCallID: chunk.ToolResult.CallID,
		})
		resIdx := len(targetSess.Messages) - 1
		resMsg := targetSess.Messages[resIdx]
		precomputeCmd = func() tea.Msg { return precomputeToolMsg{idx: resIdx, msg: resMsg} }
		if isActive {
			m.statusText = "Processing"
		}
	}

	if chunk.ReasoningContent != "" {
		if isActive {
			m.state = stateThinking
			m.statusText = "Thinking"
		}
	}

	var renderCmd tea.Cmd
	if chunk.Content != "" {
		if isActive {
			if m.state != stateGenerating {
				m.state = stateGenerating
				m.stateStart = time.Now()
			}
			m.statusText = "Generating"
		}
		msgs := targetSess.Messages
		aiIdx := len(msgs) - 1
		if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage {
			targetSess.Messages[len(msgs)-1].Content += chunk.Content
		} else {
			targetSess.Messages = append(targetSess.Messages, types.Message{
				Type:    types.AIMessage,
				Content: chunk.Content,
			})
			aiIdx = len(targetSess.Messages) - 1
		}

		if isActive {
			if !m.isAIRendering {
				m.isAIRendering = true
				m.pendingAIRender = false
				viewportWidth := max(10, m.viewport.Width)
				latestContent := targetSess.Messages[aiIdx].Content
				renderCmd = renderAIMessageCmd(sessID, aiIdx, latestContent, viewportWidth)
			} else {
				m.pendingAIRender = true
			}
		}
	}

	if isActive && (chunk.ToolCall != nil || chunk.ToolResult != nil) {
		m = m.updateViewportContent()
	}
	return m, tea.Batch(waitForNextChunk(sessID, m.chunkChan), renderCmd, precomputeCmd)
}

func (m Model) handleAgentFinished(sessID string) (tea.Model, tea.Cmd) {
	targetSess := m.getSessionByID(sessID)
	if targetSess != nil {
		_ = targetSess.SaveConversation()
	}

	if m.session != nil && m.session.ID == sessID {
		m.state = stateInput
		m.statusText = ""
		m.input.Model.Focus()
		m.tokenCount = token.CountTokens(m.session.GetPrompt())
		newModel, renderCmd := m.finalizeAIMessageRender(sessID)
		return newModel, renderCmd
	}
	return m, nil
}

func (m Model) handleAgentError(sessID string, err error) (tea.Model, tea.Cmd) {
	targetSess := m.getSessionByID(sessID)
	if targetSess != nil {
		targetSess.Messages = append(targetSess.Messages, types.Message{
			Type:    types.CommandErrorResultMessage,
			Content: fmt.Sprintf("Error: %v", err),
		})
		_ = targetSess.SaveConversation()
	}

	if m.session != nil && m.session.ID == sessID {
		m.state = stateInput
		m.statusText = ""
		m.isAIRendering = false
		m.pendingAIRender = false
		m.input.Model.Focus()
		return m.updateViewportContent(), nil
	}
	return m, nil
}

func (m Model) updateViewportContent() Model {
	wasAtBottom := m.viewport.AtBottom()
	trailing := ""
	if m.state != stateInput {
		text := "Thinking "
		if m.state == stateGenerating {
			text = "Generating "
		}
		trailing = core.RenderThinkingSpinner(text, m.spinner.View())
	}
	isStreaming := m.state == stateGenerating || m.state == stateThinking
	m.viewport.UpdateContent(m.session.Messages, isStreaming, trailing)
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
	return m
}

func (m Model) updateLayout() Model {
	if m.width <= 0 || m.height <= 0 {
		return m
	}

	m.input.SetWidth(m.width)
	m.input.UpdateHeight(m.height)

	inputHeight := m.input.Model.Height() + core.TextAreaContainerStyle.GetVerticalFrameSize()
	statusHeight := lipgloss.Height(m.statusView())
	viewportHeight := max(1, m.height-inputHeight-statusHeight)
	viewportWidth := max(10, m.width)

	if m.viewport.Height != viewportHeight || m.viewport.Width != viewportWidth {
		m.viewport.Resize(viewportWidth, viewportHeight)
		if m.session != nil {
			m = m.updateViewportContent()
		}
	}

	modalWidth := min(90, max(50, m.width-4))
	modalHeight := min(30, max(12, m.height-4))
	m.selector.Width = modalWidth
	m.selector.Height = modalHeight
	m.selector.SearchInput.Width = modalWidth - 20
	return m
}

func waitForNextChunk(sessID string, ch chan coagent.AgentStreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return agentFinishedMsg{sessID: sessID}
		}
		return streamChunkMsg{sessID: sessID, chunk: chunk}
	}
}

func renderAIMessageCmd(sessID string, msgIdx int, content string, width int) tea.Cmd {
	return func() tea.Msg {
		if content == "" {
			return aiRenderedMsg{
				sessID:  sessID,
				msgIdx:  msgIdx,
				content: content,
				lines:   nil,
				width:   width,
			}
		}

		lines, err := markdown.RenderLines(content, width)
		if err != nil {
			lines = strings.Split(content, "\n")
		}
		return aiRenderedMsg{
			sessID:  sessID,
			msgIdx:  msgIdx,
			content: content,
			lines:   lines,
			width:   width,
		}
	}
}

func (m Model) renderLastAIMessage(sessID string) (Model, tea.Cmd) {
	if m.session == nil || m.session.ID != sessID {
		return m, nil
	}
	messages := m.session.Messages
	lastIdx := len(messages) - 1
	if lastIdx < 0 || messages[lastIdx].Type != types.AIMessage {
		return m, nil
	}

	m.pendingAIRender = false
	m.isAIRendering = true
	viewportWidth := max(10, m.viewport.Width)
	return m, renderAIMessageCmd(sessID, lastIdx, messages[lastIdx].Content, viewportWidth)
}

func (m Model) finalizeAIMessageRender(sessID string) (Model, tea.Cmd) {
	if m.isAIRendering {
		m.pendingAIRender = true
		return m, nil
	}

	if m.session == nil || m.session.ID != sessID {
		return m, nil
	}

	messages := m.session.Messages
	lastIdx := len(messages) - 1
	if lastIdx < 0 || messages[lastIdx].Type != types.AIMessage {
		return m.updateViewportContent(), nil
	}

	return m.renderLastAIMessage(sessID)
}

func editInEditorCmd(content string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	tmpfile, err := os.CreateTemp("", "co-prompt-*.md")
	if err != nil {
		return func() tea.Msg { return editorFinishedMsg{err: err} }
	}
	_ = os.WriteFile(tmpfile.Name(), []byte(content), 0644)
	tmpfile.Close()

	c := exec.Command(editor, tmpfile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpfile.Name())
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		data, readErr := os.ReadFile(tmpfile.Name())
		return editorFinishedMsg{content: strings.TrimSpace(string(data)), err: readErr}
	})
}

func animateTitleTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return animateTitleTickMsg{}
	})
}

func clearStatusBarCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return clearStatusBarMsg{}
	})
}
