package coagentui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/types"
	coderui "github.com/sokinpui/coder/internal/ui/coder"
	"github.com/sokinpui/coder/internal/ui/core"
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
			return m.updateViewportContent(), nil
		}
		m.input.Model.InsertString(msg.Content)
		return m.updateLayout(), nil
	case ctrlCTimeoutMsg:
		m.ctrlCPressed = false
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.viewport.Resize(m.width, m.height)
		m = m.updateLayout()
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

	m.session.Messages = append(m.session.Messages, types.Message{
		Type:    types.UserMessage,
		Content: prompt,
	})
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
		m.session.Runtime.AgentLoop(ctx, "", preparedMessages, m.chunkChan)
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
	}

	if chunk.ToolResult != nil {
		targetSess.Messages = append(targetSess.Messages, types.Message{
			Type:       types.ToolResultMessage,
			Content:    chunk.ToolResult.Output,
			ToolCallID: chunk.ToolResult.CallID,
		})
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

	if chunk.Content != "" {
		if isActive {
			if m.state != stateGenerating {
				m.state = stateGenerating
				m.stateStart = time.Now()
			}
			m.statusText = "Generating"
		}
		msgs := targetSess.Messages
		if len(msgs) > 0 && msgs[len(msgs)-1].Type == types.AIMessage {
			targetSess.Messages[len(msgs)-1].Content += chunk.Content
		} else {
			targetSess.Messages = append(targetSess.Messages, types.Message{
				Type:    types.AIMessage,
				Content: chunk.Content,
			})
		}
	}

	if isActive {
		m = m.updateViewportContent()
	}
	return m, waitForNextChunk(sessID, m.chunkChan)
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
		m.tokenCount = token.CountTokens(m.session.Messages)
		return m.updateViewportContent(), nil
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
	m.viewport.UpdateContent(m.session.Messages, m.state == stateGenerating, trailing)
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

	m.viewport.Resize(m.width, viewportHeight)
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
