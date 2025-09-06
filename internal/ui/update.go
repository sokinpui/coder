package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateThinking, stateGenerating:
			switch msg.Type {
			case tea.KeyCtrlC:
				if m.cancelGeneration != nil {
					m.cancelGeneration()
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		case stateIdle:
			switch msg.Type {
			case tea.KeyCtrlC:
				m.quitting = true
				return m, tea.Quit
			case tea.KeyCtrlJ:
				input := m.textArea.Value()
				if strings.TrimSpace(input) == "" {
					return m, nil
				}
				m.state = stateThinking
				m.isStreaming = true
				m.streamSub = make(chan string)
				m.textArea.Blur()

				ctx, cancel := context.WithCancel(context.Background())
				m.cancelGeneration = cancel

				go m.generator.GenerateTask(ctx, input, m.streamSub)

				m.messages = append(m.messages, message{isUser: true, content: input})
				m.messages = append(m.messages, message{isUser: false, content: ""}) // Placeholder for AI
				m.lastRenderedAIPart = ""

				m.viewport.SetContent(m.renderConversation())
				m.viewport.GotoBottom()
				m.textArea.Reset()
				m.textArea.SetHeight(1)

				return m, tea.Batch(listenForStream(m.streamSub), m.spinner.Tick)
			}
		}

	case spinner.TickMsg:
		if m.state != stateThinking {
			return m, nil
		}

		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)

		renderedOutput := m.renderConversation()
		m.viewport.SetContent(renderedOutput + m.spinner.View() + " Thinking...")
		m.viewport.GotoBottom()

		return m, spinnerCmd

	case streamResultMsg:
		lastMsg := &m.messages[len(m.messages)-1]
		if m.state == stateThinking {
			m.state = stateGenerating
			lastMsg.content += string(msg)
			return m, tea.Batch(listenForStream(m.streamSub), renderTick())
		}
		lastMsg.content += string(msg)
		return m, listenForStream(m.streamSub)

	case streamFinishedMsg:
		m.isStreaming = false
		m.viewport.SetContent(m.renderConversation())

		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil

	case renderTickMsg:
		if m.state != stateGenerating || !m.isStreaming {
			return m, nil
		}

		lastMsg := m.messages[len(m.messages)-1]
		if lastMsg.content != m.lastRenderedAIPart {
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			m.lastRenderedAIPart = lastMsg.content
		}

		return m, renderTick()

	case errorMsg:
		m.isStreaming = false

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		m.messages[len(m.messages)-1].content = errorContent

		m.viewport.SetContent(m.renderConversation())

		m.viewport.GotoBottom()
		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.textArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalPadding())
		m.viewport.Width = msg.Width

		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.viewport.Width),
		)
		if err == nil {
			m.glamourRenderer = renderer
			m.viewport.SetContent(m.renderConversation())
		}
	}

	// Handle updates for textarea and viewport based on focus.
	isRuneKey := false
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyRunes {
		isRuneKey = true
	}

	// When the textarea is focused, it gets all messages.
	// The viewport only gets messages that are not character runes.
	if m.textArea.Focused() {
		m.textArea, cmd = m.textArea.Update(msg)
		cmds = append(cmds, cmd)

		if !isRuneKey {
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	} else {
		// When the textarea is not focused (e.g., during generation),
		// the viewport gets all messages.
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	inputHeight := min(m.textArea.LineCount(), m.height/4) + 1
	m.textArea.SetHeight(inputHeight)

	helpViewHeight := lipgloss.Height(m.helpView())
	viewportHeight := m.height - m.textArea.Height() - helpViewHeight - textAreaStyle.GetVerticalPadding() - 2
	if viewportHeight < 0 {
		viewportHeight = 0
	}
	m.viewport.Height = viewportHeight

	return m, tea.Batch(cmds...)
}
