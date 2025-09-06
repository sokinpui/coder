package ui

import (
	"coder/internal/config"
	"coder/internal/generation"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
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
	generator        *generation.Generator
	streamSub        chan string
	cancelGeneration context.CancelFunc
	conversation     string
	err              error
	state            state
	quitting         bool
	screenHeight     int
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
	ta.SetWidth(80 - textAreaStyle.GetHorizontalPadding())
	ta.SetHeight(1)
	ta.MaxHeight = 0
	ta.MaxWidth = 0
	ta.Prompt = ""
	ta.ShowLineNumbers = false

	return Model{
		textArea:  ta,
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
			case tea.KeyCtrlL:
				m.conversation = ""
				m.err = nil
				return m, nil
			case tea.KeyCtrlJ:
				input := m.textArea.Value()
				if strings.TrimSpace(input) == "" {
					return m, nil
				}

				m.state = stateGenerating
				m.err = nil
				m.streamSub = make(chan string)
				m.textArea.Blur()

				ctx, cancel := context.WithCancel(context.Background())
				m.cancelGeneration = cancel

				go m.generator.GenerateTask(ctx, input, m.streamSub)

				m.conversation += submittedInputStyle.Render(fmt.Sprintf("\nYou\n%s\n\n", input))
				m.conversation += outputStyle.Render("AI\n")

				return m, listenForStream(m.streamSub)
			}
		}

	case streamResultMsg:
		m.conversation += outputStyle.Render(string(msg))
		return m, listenForStream(m.streamSub)

	case streamFinishedMsg:
		m.conversation += "\n"
		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil

	case errorMsg:
		m.err = msg.error
		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, nil
	case tea.WindowSizeMsg:
		m.textArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalPadding())
		m.screenHeight = msg.Height
	}

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	inputHeight := min(m.textArea.LineCount(), m.screenHeight/4) + 1
	m.textArea.SetHeight(inputHeight)

	return m, tea.Batch(cmds...)
}

// View renders the program's UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var ui strings.Builder

	ui.WriteString(m.conversation)

	if m.err != nil {
		ui.WriteString(errorStyle.Render(fmt.Sprintf("\nError: %v\n", m.err)))
	}

	help := helpStyle.Render("Press Ctrl+J to submit, Ctrl+C to quit")
	if m.state == stateGenerating {
		help = generatingHelpStyle.Render("Generating... Press Ctrl+C to quit")
	}

	ui.WriteString(fmt.Sprintf("\n%s\n%s",
		textAreaStyle.Render(m.textArea.View()),
		help))

	return ui.String()
}
