package coagentui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
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
		m.height = msg.Height
		m.ready = true
		m = m.updateLayout()
		return m.updateViewportContent(), nil
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
			m = m.flushPendingText()
			m.history = append(m.history, historyItem{
				kind: kindNote,
				text: "[Interrupted by user]",
			})
			return m.updateViewportContent(), textinput.Blink
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

	case tea.KeyPgUp:
		m.viewport.HalfPageUp()
		return m, nil

	case tea.KeyPgDown:
		m.viewport.HalfPageDown()
		return m, nil

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

	switch msg.String() {
	case "ctrl+u":
		m.viewport.HalfPageUp()
		return m, nil
	case "ctrl+d":
		if m.state == stateRunning || m.input.Value() != "" {
			m.viewport.HalfPageDown()
			return m, nil
		}
	}

	if m.state == stateRunning {
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

	m.history = append(m.history, historyItem{
		kind: kindUser,
		text: prompt,
	})
	m = m.updateViewportContent()

	m.messages = append(m.messages, types.Message{
		Type:    types.UserMessage,
		Content: prompt,
	})

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
		m = m.flushPendingText()
		m.statusText = fmt.Sprintf("Running %s...", chunk.ToolCall.Name)
		m.history = append(m.history, historyItem{
			kind: kindToolCall,
			text: formatToolCall(chunk.ToolCall),
		})
	}

	if chunk.ToolResult != nil {
		m.history = append(m.history, historyItem{
			kind: kindToolResult,
			text: formatToolResult(chunk.ToolResult),
		})
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

	m = m.updateViewportContent()
	return m, waitForNextChunk(m.chunkChan)
}

func (m Model) handleAgentFinished() (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.statusText = "Ready"
	m.input.Focus()

	m = m.flushPendingText()
	return m.updateViewportContent(), textinput.Blink
}

func (m Model) handleAgentError(err error) (tea.Model, tea.Cmd) {
	m.state = stateInput
	m.statusText = "Error"
	m.input.Focus()

	m = m.flushPendingText()
	m.history = append(m.history, historyItem{
		kind: kindError,
		text: fmt.Sprintf("Error: %v", err),
	})
	return m.updateViewportContent(), textinput.Blink
}

func (m Model) flushPendingText() Model {
	if m.reasoningText != nil && m.reasoningText.Len() > 0 {
		txt := strings.TrimSpace(m.reasoningText.String())
		m.reasoningText.Reset()
		if txt != "" {
			m.history = append(m.history, historyItem{
				kind: kindThinking,
				text: txt,
			})
		}
	}

	if m.assistantText != nil && m.assistantText.Len() > 0 {
		txt := strings.TrimSpace(m.assistantText.String())
		m.assistantText.Reset()
		if txt != "" {
			m.history = append(m.history, historyItem{
				kind: kindAssistant,
				text: txt,
			})
		}
	}
	return m
}

func (m Model) renderAllContent() string {
	var sb strings.Builder
	w := m.renderWidth()

	for _, item := range m.history {
		sb.WriteString(m.renderHistoryItem(item, w))
	}

	if m.reasoningText != nil && m.reasoningText.Len() > 0 {
		txt := strings.TrimSpace(m.reasoningText.String())
		if txt != "" {
			header := thinkingHeaderStyle.Render("✦ Thinking")
			body := reasoningStyle.Width(w).Render(txt)
			sb.WriteString(fmt.Sprintf("%s\n%s\n\n", header, body))
		}
	}

	if m.assistantText != nil && m.assistantText.Len() > 0 {
		txt := strings.TrimSpace(m.assistantText.String())
		if txt != "" {
			rendered := markdown.Render(txt, w)
			if rendered == "" {
				rendered = txt
			}
			sb.WriteString(rendered)
			sb.WriteString("\n\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func (m Model) renderHistoryItem(item historyItem, w int) string {
	switch item.kind {
	case kindUser:
		return fmt.Sprintf("%s %s\n\n", userHeaderStyle.Render("❯"), item.text)
	case kindThinking:
		header := thinkingHeaderStyle.Render("✦ Thinking")
		body := reasoningStyle.Width(w).Render(item.text)
		return fmt.Sprintf("%s\n%s\n\n", header, body)
	case kindAssistant:
		rendered := markdown.Render(item.text, w)
		if rendered == "" {
			rendered = item.text
		}
		return rendered + "\n\n"
	case kindToolCall:
		return item.text + "\n"
	case kindToolResult:
		return item.text + "\n\n"
	case kindNote:
		return systemNoteStyle.Render(item.text) + "\n\n"
	case kindError:
		return toolErrorStyle.Render(item.text) + "\n\n"
	default:
		return ""
	}
}

func (m Model) updateViewportContent() Model {
	wasAtBottom := m.viewport.AtBottom()
	m.viewport.SetContent(m.renderAllContent())
	if wasAtBottom {
		m.viewport.GotoBottom()
	}
	return m
}

func (m Model) updateLayout() Model {
	if m.width <= 0 || m.height <= 0 {
		return m
	}

	inputLine := promptPrefixStyle.Render("❯ ") + m.input.View()
	inputBox := inputContainerStyle.Width(max(10, m.width-2)).Render(inputLine)
	statusBar := m.statusView()

	inputHeight := lipgloss.Height(inputBox)
	statusHeight := lipgloss.Height(statusBar)
	viewportHeight := max(1, m.height-inputHeight-statusHeight)

	m.viewport.Width = m.width
	m.viewport.Height = viewportHeight
	m.input.Width = max(10, m.width-6)
	return m
}

func (m Model) statusView() string {
	if m.state == stateRunning {
		status := m.statusText
		if status == "" {
			status = "Agent working..."
		}
		left := fmt.Sprintf("%s %s", m.spinner.View(), spinnerStyle.Render(status))
		right := toolMutedStyle.Render("(Ctrl+C to interrupt)")
		spacing := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right))
		return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", spacing), right)
	}

	left := statusBarStyle.Render("Co Agent")
	if m.cfg != nil && m.cfg.Generation.ModelCode != "" {
		left = statusBarStyle.Render(fmt.Sprintf("Co Agent (%s)", m.cfg.Generation.ModelCode))
	}
	right := toolMutedStyle.Render("Ctrl+C to quit")
	spacing := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", spacing), right)
}

func (m Model) renderWidth() int {
	if m.width > 0 {
		return max(20, m.width-4)
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

func formatToolCall(call *coagent.ToolCallInfo) string {
	if call == nil {
		return ""
	}
	summary := summarizeToolArgs(call.Arguments)
	prefix := toolCallStyle.Render("⚡ " + call.Name)
	if summary == "" {
		return prefix + "()"
	}
	return fmt.Sprintf("%s(%s)", prefix, toolMutedStyle.Render(summary))
}

func summarizeToolArgs(args string) string {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(trimmed), &m); err == nil {
		for _, key := range []string{"path", "command", "query"} {
			if val, ok := m[key]; ok {
				if s, ok := val.(string); ok && s != "" {
					return truncateSingleLine(s, 60)
				}
			}
		}
	}
	return truncateSingleLine(trimmed, 60)
}

func formatToolResult(result *coagent.ToolResultInfo) string {
	if result == nil {
		return ""
	}
	output := strings.TrimSpace(result.Output)
	if output == "" || output == "(no output)" {
		return fmt.Sprintf("↳ %s", toolMutedStyle.Render("(no output)"))
	}
	firstLine := strings.Split(output, "\n")[0]
	summary := truncateSingleLine(firstLine, 80)
	if strings.HasPrefix(output, "Error:") || strings.Contains(output, "Command exited with error:") {
		clean := strings.TrimPrefix(summary, "Error: ")
		return fmt.Sprintf("↳ %s %s", toolErrorStyle.Render("✗"), toolResultStyle.Render(clean))
	}
	return fmt.Sprintf("↳ %s %s", toolSuccessStyle.Render("✓"), toolResultStyle.Render(summary))
}

func truncateSingleLine(s string, maxLen int) string {
	clean := strings.Join(strings.Fields(s), " ")
	if len(clean) <= maxLen {
		return clean
	}
	return clean[:maxLen-3] + "..."
}
