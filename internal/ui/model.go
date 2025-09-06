package ui

import (
	"coder/internal/config"
	"coder/internal/generation"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
	textAreaStyle       = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
	generatingHelpStyle = helpStyle.Copy().Italic(true)
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type state int

const (
	stateIdle state = iota
	stateGenerating
)

type (
	streamResultMsg   string
	streamFinishedMsg struct{}
	errorMsg          struct{ error }
)

// listenForStream waits for the next message from the generation stream.
func listenForStream(sub chan string) tea.Cmd {
	return func() tea.Msg {
		content, ok := <-sub
		if !ok {
			return streamFinishedMsg{}
		}
		if strings.HasPrefix(content, "Error:") {
			return errorMsg{errors.New(strings.TrimSpace(strings.TrimPrefix(content, "Error:")))}
		}
		return streamResultMsg(content)
	}
}

// Model defines the state of the application's UI.
type Model struct {
	textArea         textarea.Model
	viewport         viewport.Model
	generator        *generation.Generator
	streamSub        chan string
	cancelGeneration context.CancelFunc
	conversation     string
	state            state
	quitting         bool
	height           int
	width            int
	glamourRenderer  *glamour.TermRenderer
}

// NewModel creates a new UI model.
func NewModel(cfg *config.Config) (Model, error) {
	gen, err := generation.New(cfg)
	if err != nil {
		return Model{}, err
	}

	ta := textarea.New()
	ta.Placeholder = "Enter your prompt..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.MaxHeight = 0
	ta.MaxWidth = 0
	ta.Prompt = ""
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20) // Initial size, will be updated

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(vp.Width),
	)
	if err != nil {
		return Model{}, err
	}

	return Model{
		textArea:        ta,
		viewport:        vp,
		generator:       gen,
		state:           stateIdle,
		glamourRenderer: renderer,
	}, nil
}

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
		case stateGenerating:
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
				m.state = stateGenerating
				m.streamSub = make(chan string)
				m.textArea.Blur()

				ctx, cancel := context.WithCancel(context.Background())
				m.cancelGeneration = cancel

				go m.generator.GenerateTask(ctx, input, m.streamSub)

				m.conversation += fmt.Sprintf("\n**You**\n\n%s\n\n**AI**\n\n", input)
				renderedOutput, err := m.glamourRenderer.Render(m.conversation)
				if err != nil {
					renderedOutput = m.conversation
				}
				m.viewport.SetContent(renderedOutput)
				m.viewport.GotoBottom()

				m.textArea.Reset()

				return m, listenForStream(m.streamSub)
			}
		}

	case streamResultMsg:
		m.conversation += string(msg)
		renderedOutput, err := m.glamourRenderer.Render(m.conversation)
		if err != nil {
			renderedOutput = m.conversation
		}
		m.viewport.SetContent(renderedOutput)
		m.viewport.GotoBottom()
		return m, listenForStream(m.streamSub)

	case streamFinishedMsg:
		m.conversation += "\n"
		renderedOutput, err := m.glamourRenderer.Render(m.conversation)
		if err != nil {
			renderedOutput = m.conversation
		}
		m.viewport.SetContent(renderedOutput)

		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil

	case errorMsg:
		m.conversation += fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		renderedOutput, err := m.glamourRenderer.Render(m.conversation)
		if err != nil {
			renderedOutput = m.conversation
		}
		m.viewport.SetContent(renderedOutput)
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
			renderedOutput, err := m.glamourRenderer.Render(m.conversation)
			if err == nil {
				m.viewport.SetContent(renderedOutput)
			}
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

// helpView renders the help text.
func (m Model) helpView() string {
	help := helpStyle.Render("Ctrl+J to submit, Ctrl+C to quit")
	if m.state == stateGenerating {
		help = generatingHelpStyle.Render("Generating... Ctrl+C to quit")
	}
	return help
}

// View renders the program's UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	return fmt.Sprintf("%s\n%s\n%s",
		m.viewport.View(),
		textAreaStyle.Render(m.textArea.View()),
		m.helpView(),
	)
}
