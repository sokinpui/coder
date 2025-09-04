package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model holds the state of the Bubble Tea application.
type Model struct {
	textInput textinput.Model // The input box for user code.
	outputLog []string        // Stores previous inputs and generated outputs.
	quitting  bool            // Indicates if the application is in the process of quitting.
}

// NewModel creates a new instance of the Model.
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter your code..."
	ti.Focus()
	ti.CharLimit = 0 // No character limit
	ti.Width = 80   // Default width
	ti.Prompt = "> "

	return Model{
		textInput: ti,
		outputLog: []string{},
	}
}

// Init initializes the Bubble Tea program.
// It returns a command to start the text input blinking.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model's state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter: // Ctrl-J sends a line feed, which Bubble Tea interprets as KeyEnter
			input := m.textInput.Value()
			if strings.TrimSpace(input) == "" {
				// Don't process empty input, just clear the field.
				m.textInput.SetValue("")
				return m, nil
			}

			// Add the user's input to the log.
			m.outputLog = append(m.outputLog, inputStyle.Render(fmt.Sprintf("You entered: %s", input)))

			// Placeholder for AI response.
			charCount := len(input)
			m.outputLog = append(m.outputLog, outputStyle.Render(fmt.Sprintf("output: You input %d char", charCount)))
			m.outputLog = append(m.outputLog, "") // Add a blank line for readability

			m.textInput.SetValue("") // Clear the input field after submission
			return m, nil            // No further command needed, just update the model
		}

	case tea.WindowSizeMsg:
		// Adjust text input width based on terminal size.
		m.textInput.Width = msg.Width - len(m.textInput.Prompt) - 2 // Account for prompt and padding
	}

	// Always update the text input component with any other messages.
	m.textInput, cmd = m.textInput.Update(msg)
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
	s.WriteString(m.textInput.View())
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Press Ctrl+J to submit, Ctrl+C to quit"))

	return s.String()
}

var (
	inputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12")) // Blue
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")) // Green
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray
)
