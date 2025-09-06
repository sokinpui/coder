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
	"github.com/charmbracelet/lipgloss"
)

var (
	submittedInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // Gray
	outputStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))  // Green
	errorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
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

	return Model{
		textArea:  ta,
		viewport:  vp,
		generator: gen,
		state:     stateIdle,
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

				m.conversation += submittedInputStyle.Render(fmt.Sprintf("\nYou\n%s\n\n", input))
				m.conversation += outputStyle.Render("AI\n")
				m.viewport.SetContent(m.conversation)
				m.viewport.GotoBottom()

				m.textArea.Reset()

				return m, listenForStream(m.streamSub)
			}
		}

	case streamResultMsg:
		m.conversation += outputStyle.Render(string(msg))
		m.viewport.SetContent(m.conversation)
		m.viewport.GotoBottom()
		return m, listenForStream(m.streamSub)

	case streamFinishedMsg:
		m.conversation += "\n"
		m.viewport.SetContent(m.conversation)
		m.viewport.GotoBottom()
		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil

	case errorMsg:
		m.conversation += errorStyle.Render(fmt.Sprintf("\nError: %v\n", msg.error))
		m.viewport.SetContent(m.conversation)
		m.viewport.GotoBottom()
		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.textArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalPadding())
		m.viewport.Width = msg.Width
	}

	if m.state == stateIdle {
		m.textArea, cmd = m.textArea.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

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
