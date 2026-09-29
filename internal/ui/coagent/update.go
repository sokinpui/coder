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
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/core"
)

type editorFinishedMsg struct {
	content string
	err     error
}

type ctrlCTimeoutMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initPromptMsg:
		return m.startPrompt(string(msg))
	case tea.KeyMsg:
		return m.handleKey(msg)
	case streamChunkMsg:
		return m.handleStreamChunk(coagent.AgentStreamChunk(msg))
	case agentFinishedMsg:
		return m.handleAgentFinished()
	case agentErrorMsg:
		return m.handleAgentError(msg.err)
	case editorFinishedMsg:
		if msg.err == nil && msg.content != "" {
			m.input.Model.Reset()
			return m.startPrompt(msg.content)
		}
		return m, nil
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

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type != tea.KeyCtrlC {
		m.ctrlCPressed = false
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		if m.state != stateInput {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			m.state = stateInput
			m.input.Model.Focus()
			m.messages = append(m.messages, types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: "[Interrupted by user]",
			})
			return m.updateViewportContent(), nil
		}
		if m.input.Model.Value() != "" {
			m.input.Model.Reset()
			return m, nil
		}
		if m.ctrlCPressed {
			return m, tea.Quit
		}
		m.ctrlCPressed = true
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return ctrlCTimeoutMsg{}
		})
		return m, tea.Quit

	case tea.KeyCtrlD:
		if m.state == stateInput && m.input.Model.Value() == "" {
			return m, tea.Quit
		}

	case tea.KeyPgUp:
		m.viewport.HalfPageUp()
		return m, nil

	case tea.KeyPgDown:
		m.viewport.HalfPageDown()
		return m, nil

	case tea.KeyEnter:
		if m.state == stateInput {
			text := strings.TrimSpace(m.input.Model.Value())
			if text == "" {
				return m, nil
			}
			if text == "/exit" || text == "/quit" || text == "exit" || text == "quit" {
				return m, tea.Quit
			}
			m.input.Model.Reset()
			m = m.updateLayout()
			return m.startPrompt(text)
		}
	}

	switch msg.String() {
	case "ctrl+e":
		if m.state == stateInput {
			return m, editInEditorCmd(m.input.Model.Value())
		}
	case "ctrl+j":
		if m.state == stateInput {
			text := strings.TrimSpace(m.input.Model.Value())
			if text != "" {
				m.input.Model.Reset()
				m = m.updateLayout()
				return m.startPrompt(text)
			}
		}
	case "ctrl+u":
		m.viewport.HalfPageUp()
		return m, nil
	case "ctrl+d":
		m.viewport.HalfPageDown()
		return m, nil
	}

	if m.state != stateInput {
		switch msg.Type {
		case tea.KeyUp:
			m.viewport.LineUp(1)
		case tea.KeyDown:
			m.viewport.LineDown(1)
		}
		return m, nil
	}

	if m.state == stateInput {
		var cmd tea.Cmd
		m.input.Model, cmd = m.input.Model.Update(msg)
		m = m.updateLayout()
		return m, cmd
	}

	return m, nil
}

func (m Model) startPrompt(prompt string) (tea.Model, tea.Cmd) {
	m.state = stateThinking
	m.stateStart = time.Now()
	m.statusText = "Thinking"

	m.messages = append(m.messages, types.Message{
		Type:    types.UserMessage,
		Content: prompt,
	})
	m = m.updateViewportContent()

	m.chunkChan = make(chan coagent.AgentStreamChunk, 100)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	go func() {
		m.runtime.AgentLoop(ctx, "", m.messages, m.chunkChan)
	}()

	return m, tea.Batch(
		m.spinner.Tick,
		waitForNextChunk(m.chunkChan),
	)
}

func (m Model) handleStreamChunk(chunk coagent.AgentStreamChunk) (tea.Model, tea.Cmd) {
	if len(chunk.Messages) > 0 {
		m.messages = chunk.Messages
	}

	if chunk.ToolCall != nil {
		m.statusText = fmt.Sprintf("Running %s", chunk.ToolCall.Name)
		m.messages = append(m.messages, types.Message{
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
		m.messages = append(m.messages, types.Message{
			Type:       types.ToolResultMessage,
			Content:    chunk.ToolResult.Output,
			ToolCallID: chunk.ToolResult.CallID,
		})
		m.statusText = "Processing"
	}

	if chunk.ReasoningContent != "" {
		m.state = stateThinking
		m.statusText = "Thinking"
	}

	if chunk.Content != "" {
		if m.state != stateGenerating {
			m.state = stateGenerating
			m.stateStart = time.Now()
		}
		m.statusText = "Generating"
		if len(m.messages) > 0 && m.messages[len(m.messages)-1].Type == types.AIMessage {
			m.messages[len(m.messages)-1].Content += chunk.Content
		} else {
			m.messages = append(m.messages, types.Message{
				Type:    types.AIMessage,
				Content: chunk.Content,
			})
		}
	}

	m = m.updateViewportContent()
	return m, waitForNextChunk(m.chunkChan)
}

func (m Model) handleAgentFinished() (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.statusText = ""
	m.input.Model.Focus()
	m.tokenCount = token.CountTokens(m.messages)
	return m.updateViewportContent(), nil
}

func (m Model) handleAgentError(err error) (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.statusText = ""
	m.input.Model.Focus()
	m.messages = append(m.messages, types.Message{
		Type:    types.CommandErrorResultMessage,
		Content: fmt.Sprintf("Error: %v", err),
	})
	return m.updateViewportContent(), nil
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
	m.viewport.UpdateContent(m.messages, m.state == stateGenerating, trailing)
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

func waitForNextChunk(ch chan coagent.AgentStreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return agentFinishedMsg{}
		}
		return streamChunkMsg(chunk)
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
