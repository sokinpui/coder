package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model holds the state of the Bubble Tea application.
type Model struct {
	textArea  textarea.Model // The multi-line input box for user code.
	outputLog []string       // Stores previous inputs and generated outputs.
	quitting  bool           // Indicates if the application is in the process of quitting.
}

// NewModel creates a new instance of the Model.
func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your code..."
	ta.Focus()
	ta.CharLimit = 0 // No character limit
	ta.SetWidth(80)  // Default width
	ta.SetHeight(10) // Default height
	ta.Prompt = "> "
	ta.ShowLineNumbers = false // No line numbers for now, can be configured

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
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Ctrl+C or Esc to quit the application.
			m.quitting = true
			return m, tea.Quit
		case tea.KeyCtrlJ: // Ctrl+J to submit the current input.
			input := m.textArea.Value()
			if strings.TrimSpace(input) == "" {
				// Don't process empty input, just clear the field.
				m.textArea.SetValue("")
				return m, nil
			}

			// Add the user's input to the log.
			m.outputLog = append(m.outputLog, inputStyle.Render(fmt.Sprintf("You entered: %s", input)))

			// Placeholder for AI response.
			charCount := len(input)
			m.outputLog = append(m.outputLog, outputStyle.Render(fmt.Sprintf("output: You input %d char", charCount)))
			m.outputLog = append(m.outputLog, "") // Add a blank line for readability

			m.textArea.SetValue("") // Clear the input field after submission.
			return m, nil           // No further command needed, just update the model.
		}

	case tea.WindowSizeMsg:
		// Adjust textarea width based on terminal size.
		// Textarea automatically handles its height and soft-wrapping based on width.
		m.textArea.SetWidth(msg.Width - len(m.textArea.Prompt) - 2) // Account for prompt and padding.
	}

	// Always update the text input component with any other messages.
	m.textArea, cmd = m.textArea.Update(msg)
	return m, cmd
}

// View renders the program's UI.
func (m Model) View() string {
	if m.quitting {
		return "Exiting coder...\n"
	}

	var s strings.Builder

	// Display the historical output.
	for _, line := range m.outputLog {
		s.WriteString(line)
		s.WriteString("\n")
	}

	// Add the prompt for the next input.
	s.WriteString("\n")
	s.WriteString(m.textArea.View())
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Press Ctrl+J to submit, Ctrl+C to quit"))

	return s.String()
}

var (
	inputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))  // Blue
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))  // Green
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
)
