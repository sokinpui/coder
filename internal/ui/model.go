package ui

import (
	"fmt"
	"strings"
	"coder/internal/generation"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const ()

var (
	submittedInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // Gray
	outputStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))  // Green
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

type (
	streamResultMsg   string
	streamFinishedMsg struct{}
)

func waitForStreamActivity(sub chan string) tea.Cmd {
	return func() tea.Msg {
		content, ok := <-sub
		if !ok {
			return streamFinishedMsg{}
		}
		return streamResultMsg(content)
	}
}

type Model struct {
	textArea      textarea.Model
	quitting      bool
	screenHeight  int
	generator     *generation.Generator
	generating    bool
	streamSub     chan string
}

func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your code..."
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
		generator: generation.New(),
	}
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
		if m.generating {
			switch msg.Type {
			case tea.KeyCtrlC:
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlL:
			return m, tea.ClearScreen
		case tea.KeyCtrlJ:
			input := m.textArea.Value()
			if strings.TrimSpace(input) == "" {
				return m, nil
			}

			m.generating = true
			m.streamSub = make(chan string)
			m.textArea.Blur()

			go m.generator.GenerateTask(input, m.streamSub)

			output := submittedInputStyle.Render(fmt.Sprintf("You\n%s", input))
			aiHeader := outputStyle.Render("✦")

			return m, tea.Batch(
				tea.Printf("\n%s\n%s", output, aiHeader),
				waitForStreamActivity(m.streamSub),
			)
		}

	case streamResultMsg:
		cmd := tea.Printf(outputStyle.Render(string(msg)))
		return m, tea.Batch(cmd, waitForStreamActivity(m.streamSub))

	case streamFinishedMsg:
		m.generating = false
		m.streamSub = nil
		m.textArea.Reset()
		m.textArea.Focus()
		return m, tea.Printf("\n")

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

	help := helpStyle.Render("Press Ctrl+J to submit, Ctrl+C to quit")
	if m.generating {
		help = generatingHelpStyle.Render("Generating... Press Ctrl+C to quit")
	}

	return fmt.Sprintf("%s\n%s",
		textAreaStyle.Render(m.textArea.View()),
		help,
	)
}
