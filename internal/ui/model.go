package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	fixedTextareaHeight = 0
	maxLines            = 10
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))  // Blue
	outputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))  // Green
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
	textAreaStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
)

// Model holds the state of the Bubble Tea application.
type Model struct {
	textArea  textarea.Model // The multi-line input box for user code.
	outputLog []string // Stores previous inputs and generated outputs.
	quitting  bool     // Indicates if the application is in the process of quitting.
}

// NewModel creates a new instance of the Model.
func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your code..."
	ta.Focus()
	ta.CharLimit = 0 // No character limit
	ta.SetWidth(80 - textAreaStyle.GetHorizontalPadding())
	ta.SetHeight(fixedTextareaHeight - textAreaStyle.GetVerticalPadding())
	ta.Prompt = ""
	ta.ShowLineNumbers = false

	return Model{
		textArea:  ta,
		outputLog: []string{},
	}
}

// Init initializes the Bubble Tea program.
// It returns a command to start the text input blinking.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages and updates the model's state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			// Ctrl+C or Esc to quit the application.
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlJ: // Ctrl+J to submit the current input.
			input := m.textArea.Value()
			if strings.TrimSpace(input) == "" {
				// Don't process empty input, just clear the field.
				m.textArea.Reset()
				return m, nil
			}

			// Add the user's input to the log.
			m.outputLog = append(m.outputLog, inputStyle.Render(fmt.Sprintf("You entered: %s", input)))

			// Placeholder for AI response.
			charCount := len(input)
			m.outputLog = append(m.outputLog, outputStyle.Render(fmt.Sprintf("output: You input %d char", charCount)))
			m.outputLog = append(m.outputLog, "") // Add a blank line for readability

			m.textArea.Reset() // Clear the input field after submission.
			return m, nil      // No further command needed, just update the model.
		}

	case tea.WindowSizeMsg:
		m.textArea.SetWidth(msg.Width - textAreaStyle.GetHorizontalPadding())
	}

	// Always update the text input component with any other messages.
	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	lineCount := m.textArea.LineCount()
	if lineCount < maxLines {
		m.textArea.SetHeight(lineCount + 1)
	} else {
		m.textArea.SetHeight(maxLines)
	}

	return m, tea.Batch(cmds...)
}

// View renders the program's UI.
func (m Model) View() string {
	if m.quitting {
		// Why: On exit, we want to display the final output log.
		// This is particularly useful when not using an alternate screen,
		// as it leaves the conversation history in the terminal.
		return strings.Join(m.outputLog, "\n") + "\n"
	}

	// Why: We want to create a fluid, terminal-like experience where the output
	// flows above the input box. A strings.Builder is efficient for this.
	var view strings.Builder

	// Only write the output log if it's not empty, to avoid a leading newline
	// at the beginning of the session.
	if len(m.outputLog) > 0 {
		view.WriteString(strings.Join(m.outputLog, "\n") + "\n")
	}

	view.WriteString(textAreaStyle.Render(m.textArea.View()))
	view.WriteString("\n" + helpStyle.Render("Press Ctrl+J to submit, Ctrl+C to quit"))

	return view.String()
}
