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
		return m, nil
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
			var cmds []tea.Cmd
			if m.partialLine != "" {
				cmds = append(cmds, tea.Println(cleanRender(m.partialLine, m.renderWidth())))
				m.partialLine = ""
				m.streamBuffer = ""
				m.currentContent = ""
			}
			cmds = append(cmds,
				tea.Println(systemNoteStyle.Render("\n[Interrupted by user]")),
				textinput.Blink,
			)
			return m, tea.Batch(cmds...)
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
	m.streamBuffer = ""
	m.partialLine = ""

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
		if m.partialLine != "" {
			cmds = append(cmds, tea.Println(cleanRender(m.partialLine, m.renderWidth())))
			m.partialLine = ""
			m.streamBuffer = ""
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
		m.streamBuffer += chunk.Content

		if !isInIncompleteBlock(m.streamBuffer) {
			if idx := strings.LastIndex(m.streamBuffer, "\n\n"); idx != -1 {
				blockToPrint := m.streamBuffer[:idx]
				m.partialLine = m.streamBuffer[idx+2:]
				m.streamBuffer = m.partialLine

				if strings.TrimSpace(blockToPrint) != "" {
					rendered := cleanRender(blockToPrint, m.renderWidth())
					if rendered != "" {
						cmds = append(cmds, tea.Println(rendered))
					}
				}
			} else if idx := strings.LastIndex(m.streamBuffer, "\n"); idx != -1 && isSimpleLine(m.streamBuffer[:idx]) {
				linesToPrint := m.streamBuffer[:idx]
				m.partialLine = m.streamBuffer[idx+1:]
				m.streamBuffer = m.partialLine

				rendered := cleanRender(linesToPrint, m.renderWidth())
				if rendered != "" {
					cmds = append(cmds, tea.Println(rendered))
				}
			}
		}
	}

	cmds = append(cmds, waitForNextChunk(m.chunkChan))
	return m, tea.Batch(cmds...)
}

func (m Model) handleAgentFinished() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.partialLine != "" || m.streamBuffer != "" {
		remaining := m.partialLine
		if remaining == "" {
			remaining = m.streamBuffer
		}
		rendered := cleanRender(remaining, m.renderWidth())
		if rendered != "" {
			cmds = append(cmds, tea.Println(rendered))
		}
		m.partialLine = ""
		m.streamBuffer = ""
		if len(m.messages) == 0 || m.messages[len(m.messages)-1].Type != types.AIMessage {
			m.messages = append(m.messages, types.Message{
				Type:    types.AIMessage,
				Content: m.currentContent,
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

func (m Model) renderWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

func cleanRender(content string, width int) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}
	rendered := markdown.Render(trimmed, width)
	return strings.Trim(rendered, "\r\n")
}

func isInIncompleteBlock(buf string) bool {
	codeFenceCount := strings.Count(buf, "```")
	if codeFenceCount%2 != 0 {
		return true
	}
	lastNewline := strings.LastIndex(buf, "\n")
	if lastNewline != -1 {
		lastLine := strings.TrimSpace(buf[lastNewline+1:])
		if strings.HasPrefix(lastLine, "|") {
			return true
		}
	}
	return false
}

func isSimpleLine(buf string) bool {
	trimmed := strings.TrimSpace(buf)
	if strings.HasPrefix(trimmed, "|") || strings.Contains(buf, "```") {
		return false
	}
	return true
}
