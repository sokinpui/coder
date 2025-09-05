package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// fixedTextareaHeight = 0
	maxLines = 10
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))  // Blue
	outputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))  // Green
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
	textAreaStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

type Model struct {
	textArea textarea.Model
	quitting bool
}

func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your code..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(80 - textAreaStyle.GetHorizontalPadding())
	// ta.SetHeight(fixedTextareaHeight - textAreaStyle.GetVerticalPadding())
	ta.SetHeight(1)
	ta.MaxHeight = 0
	ta.Prompt = ""
	ta.ShowLineNumbers = false

	return Model{
		textArea: ta,
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
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlJ: // Ctrl+J to submit the current input.
			input := m.textArea.Value()
			if strings.TrimSpace(input) == "" {
				m.textArea.Reset()
				return m, nil
			}

			// Placeholder for AI response.
			charCount := len(input)
			output := fmt.Sprintf("%s\n%s\n",
				inputStyle.Render(fmt.Sprintf("You entered: %s", input)),
				outputStyle.Render(fmt.Sprintf("output: You input %d char", charCount)),
			)

			m.textArea.Reset()
			return m, tea.Printf(output)
		}

	case tea.WindowSizeMsg:
		m.textArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalPadding())
	}

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	lineCount := m.textArea.LineCount()
	if lineCount < maxLines {
		m.textArea.SetHeight(lineCount + 1)
	} else {
		m.textArea.SetHeight(maxLines + 1)
	}

	return m, tea.Batch(cmds...)
}

// View renders the program's UI.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	return fmt.Sprintf("%s\n%s",
		textAreaStyle.Render(m.textArea.View()),
		helpStyle.Render("Press Ctrl+J to submit, Ctrl+C to quit"),
	)
}
