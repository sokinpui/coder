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
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if m.state == stateInput {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
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
			m.input.Focus()
			return m, tea.Batch(
				tea.Println(systemNoteStyle.Render("\n[Interrupted by user]")),
				textinput.Blink,
			)
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
	m.currentContent = ""

	m.messages = append(m.messages, types.Message{
		Type:    types.UserMessage,
		Content: prompt,
	})

	m.chunkChan = make(chan coagent.AgentStreamChunk, 100)
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel

	userLine := fmt.Sprintf("\n%s %s", userHeaderStyle.Render("❯"), prompt)

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
		if len(m.currentContent) > 0 {
			cmds = append(cmds, tea.Println(m.currentContent))
			m.currentContent = ""
		}
		callLine := fmt.Sprintf("⚡ %s %s(%s)", toolCallStyle.Render("Tool:"), chunk.ToolCall.Name, chunk.ToolCall.Arguments)
		cmds = append(cmds, tea.Println(callLine))
	}

	if chunk.ToolResult != nil {
		out := formatToolOutput(chunk.ToolResult.Output)
		resLine := fmt.Sprintf("↳ %s", toolResultStyle.Render(out))
		cmds = append(cmds, tea.Println(resLine))
	}

	if chunk.Content != "" {
		m.currentContent += chunk.Content
	}

	cmds = append(cmds, waitForNextChunk(m.chunkChan))
	return m, tea.Batch(cmds...)
}

func (m Model) handleAgentFinished() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if len(m.currentContent) > 0 {
		finalText := m.currentContent
		cmds = append(cmds, tea.Println(finalText))
		if len(m.messages) == 0 || m.messages[len(m.messages)-1].Type != types.AIMessage {
			m.messages = append(m.messages, types.Message{
				Type:    types.AIMessage,
				Content: finalText,
			})
		}
		m.currentContent = ""
	}

	m.state = stateInput
	m.input.Focus()
	cmds = append(cmds, textinput.Blink)
	return m, tea.Batch(cmds...)
}

func (m Model) handleAgentError(err error) (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.input.Focus()
	errLine := toolErrorStyle.Render(fmt.Sprintf("Error: %v", err))
	return m, tea.Batch(tea.Println(errLine), textinput.Blink)
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

func formatToolOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "(no output)"
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) <= 6 {
		return trimmed
	}
	head := strings.Join(lines[:5], "\n")
	return fmt.Sprintf("%s\n... [%d lines hidden]", head, len(lines)-5)
}
