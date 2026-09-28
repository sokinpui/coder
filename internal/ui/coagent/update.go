package coagentui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.Width = max(10, m.width-4)
		return m, nil
	case spinner.TickMsg:
		if m.state == stateRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.state == stateRunning {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			m.state = stateInput
			m.statusText = "Ready"
			m.input.Focus()
			var cmds []tea.Cmd
			cmds = append(cmds, m.flushPendingText()...)
			cmds = append(cmds,
				tea.Println(systemNoteStyle.Render("[Interrupted by user]\n")),
				textinput.Blink,
			)
			return m, tea.Batch(cmds...)
		}
		if m.input.Value() != "" {
			m.input.Reset()
			return m, nil
		}
		return m, tea.Quit

	case tea.KeyCtrlD:
		if m.state == stateInput && m.input.Value() == "" {
			return m, tea.Quit
		}

	case tea.KeyEnter:
		if m.state == stateInput {
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			if text == "/exit" || text == "/quit" || text == "exit" || text == "quit" {
				return m, tea.Quit
			}
			m.input.Reset()
			return m.startPrompt(text)
		}
	}

	if m.state == stateInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) startPrompt(prompt string) (tea.Model, tea.Cmd) {
	m.state = stateRunning
	m.statusText = "Thinking..."
	m.reasoningText.Reset()
	m.assistantText.Reset()
	m.pendingCalls = make(map[string]coagent.ToolCallInfo)

	m.messages = append(m.messages, types.Message{
		Type:    types.UserMessage,
		Content: prompt,
	})

	m.chunkChan = make(chan coagent.AgentStreamChunk, 100)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	userLine := fmt.Sprintf("%s %s\n", userHeaderStyle.Render("❯"), prompt)

	go func() {
		m.runtime.AgentLoop(ctx, "", m.messages, m.chunkChan)
	}()

	return m, tea.Batch(
		tea.Println(userLine),
		m.spinner.Tick,
		waitForNextChunk(m.chunkChan),
	)
}

func (m Model) handleStreamChunk(chunk coagent.AgentStreamChunk) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if len(chunk.Messages) > 0 {
		m.messages = chunk.Messages
	}

	if chunk.ToolCall != nil {
		cmds = append(cmds, m.flushPendingText()...)

		if m.pendingCalls == nil {
			m.pendingCalls = make(map[string]coagent.ToolCallInfo)
		}
		if chunk.ToolCall.CallID != "" {
			m.pendingCalls[chunk.ToolCall.CallID] = *chunk.ToolCall
		}

		m.statusText = fmt.Sprintf("Running %s...", chunk.ToolCall.Name)
		callLine := renderToolCall(chunk.ToolCall, m.renderWidth())
		cmds = append(cmds, tea.Println(callLine))
	}

	if chunk.ToolResult != nil {
		var matchedCall *coagent.ToolCallInfo
		if m.pendingCalls != nil && chunk.ToolResult.CallID != "" {
			if call, ok := m.pendingCalls[chunk.ToolResult.CallID]; ok {
				callCopy := call
				matchedCall = &callCopy
				delete(m.pendingCalls, chunk.ToolResult.CallID)
			}
		}

		resLine := renderToolResult(chunk.ToolResult, matchedCall)
		cmds = append(cmds, tea.Println(resLine+"\n"))
		m.statusText = "Processing..."
	}

	if chunk.ReasoningContent != "" {
		m.reasoningText.WriteString(chunk.ReasoningContent)
		m.statusText = "Thinking..."
	}

	if chunk.Content != "" {
		m.assistantText.WriteString(chunk.Content)
		m.statusText = "Responding..."
	}

	cmds = append(cmds, waitForNextChunk(m.chunkChan))
	return m, tea.Batch(cmds...)
}

func (m Model) handleAgentFinished() (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.statusText = "Ready"
	m.input.Focus()

	var cmds []tea.Cmd
	cmds = append(cmds, m.flushPendingText()...)
	cmds = append(cmds, textinput.Blink)
	return m, tea.Batch(cmds...)
}

func (m Model) handleAgentError(err error) (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.statusText = "Error"
	m.input.Focus()

	var cmds []tea.Cmd
	cmds = append(cmds, m.flushPendingText()...)
	cmds = append(cmds, tea.Println(toolErrorStyle.Render(fmt.Sprintf("Error: %v\n", err))))
	cmds = append(cmds, textinput.Blink)
	return m, tea.Batch(cmds...)
}

func (m *Model) flushPendingText() []tea.Cmd {
	var cmds []tea.Cmd
	if m.reasoningText != nil && m.reasoningText.Len() > 0 {
		txt := strings.TrimSpace(m.reasoningText.String())
		m.reasoningText.Reset()
		if txt != "" {
			width := m.renderWidth()
			header := thinkingHeaderStyle.Render("✦ Thinking")
			body := reasoningStyle.Width(width).Render(txt)
			cmds = append(cmds, tea.Println(fmt.Sprintf("%s\n%s\n", header, body)))
		}
	}

	if m.assistantText != nil && m.assistantText.Len() > 0 {
		txt := strings.TrimSpace(m.assistantText.String())
		m.assistantText.Reset()
		if txt != "" {
			width := m.renderWidth()
			rendered := markdown.Render(txt, width)
			if rendered == "" {
				rendered = txt
			}
			cmds = append(cmds, tea.Println(rendered+"\n"))
		}
	}
	return cmds
}

func (m Model) renderWidth() int {
	if m.width > 0 {
		return max(20, m.width-2)
	}
	return 80
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
