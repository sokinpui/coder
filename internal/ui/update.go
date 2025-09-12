package ui

import (
	"coder/internal/core"
	"log"
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
	return tea.Batch(textarea.Blink, loadInitialContextCmd())
}

func (m Model) newSession() (Model, tea.Cmd) {
	// Save the current conversation before starting a new one.
	if err := m.historyManager.SaveConversation(m.messages, m.systemInstructions, m.providedDocuments, m.projectSourceCode); err != nil {
		// Log the error, but don't block the user from starting a new session.
		log.Printf("Error saving conversation for /new command: %v", err)
	}

	// Reset the message history to the initial state.
	m.messages = []core.Message{
		{Type: core.InitMessage, Content: welcomeMessage},
	}

	// Reset UI and state flags.
	m.lastInteractionFailed = false
	m.lastRenderedAIPart = ""
	m.textArea.Reset()
	m.textArea.SetHeight(1)
	m.textArea.Focus()
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderConversation())

	// Recalculate the token count for the base context.
	initialPrompt := core.BuildPrompt(m.systemInstructions, m.providedDocuments, m.projectSourceCode, nil)
	m.isCountingTokens = true

	return m, countTokensCmd(initialPrompt)
}

func (m Model) startGeneration() (Model, tea.Cmd) {
	prompt := core.BuildPrompt(m.systemInstructions, m.providedDocuments, m.projectSourceCode, m.messages)

	m.state = stateThinking
	m.isStreaming = true
	m.isCountingTokens = true
	m.streamSub = make(chan string)
	m.textArea.Blur()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelGeneration = cancel

	go m.generator.GenerateTask(ctx, prompt, m.streamSub)
	m.messages = append(m.messages, core.Message{Type: core.AIMessage, Content: ""}) // Placeholder for AI
	m.lastRenderedAIPart = ""
	m.lastInteractionFailed = false // Reset this flag for the new generation attempt.

	m.viewport.SetContent(m.renderConversation())
	m.viewport.GotoBottom()

	return m, tea.Batch(listenForStream(m.streamSub), m.spinner.Tick, countTokensCmd(prompt))
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.textArea.Value()
	if strings.TrimSpace(input) == "" {
		return m, nil
	}

	if strings.HasPrefix(input, ":") {
		actionResult, isAction, actionSuccess := core.ProcessAction(input)
		if isAction {
			m.messages = append(m.messages, core.Message{Type: core.ActionMessage, Content: input})
			if actionSuccess {
				m.messages = append(m.messages, core.Message{Type: core.ActionResultMessage, Content: actionResult})
			} else {
				m.messages = append(m.messages, core.Message{Type: core.ActionErrorResultMessage, Content: actionResult})
			}
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			m.textArea.Reset()
			m.textArea.SetHeight(1)
			return m, nil
		}

		cmdResult, isCmd, cmdSuccess := core.ProcessCommand(input, m.messages, m.config)
		if isCmd {
			if cmdSuccess && cmdResult == core.NewSessionResult {
				return m.newSession()
			}

			if cmdSuccess && cmdResult == core.RegenerateResult {
				lastUserMsgIndex := -1
				for i := len(m.messages) - 1; i >= 0; i-- {
					if m.messages[i].Type == core.UserMessage {
						lastUserMsgIndex = i
						break
					}
				}

				if lastUserMsgIndex == -1 {
					m.messages = append(m.messages, core.Message{Type: core.CommandErrorResultMessage, Content: "No previous user prompt to regenerate from."})
					m.viewport.SetContent(m.renderConversation())
					m.viewport.GotoBottom()
					m.textArea.Reset()
					m.textArea.SetHeight(1)
					return m, nil
				}

				m.messages = m.messages[:lastUserMsgIndex+1]
				m.textArea.Reset()
				m.textArea.SetHeight(1)
				return m.startGeneration()
			}

			m.generator.Config = m.config.Generation
			m.messages = append(m.messages, core.Message{Type: core.CommandMessage, Content: input})
			if cmdSuccess {
				m.messages = append(m.messages, core.Message{Type: core.CommandResultMessage, Content: cmdResult})
			} else {
				m.messages = append(m.messages, core.Message{Type: core.CommandErrorResultMessage, Content: cmdResult})
			}
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			m.textArea.Reset()
			m.textArea.SetHeight(1)
			return m, nil
		}
	}

	// This is now the "new user prompt" section.
	if m.lastInteractionFailed {
		if len(m.messages) >= 2 {
			// Remove the previous user message and the failed/cancelled AI message
			m.messages = m.messages[:len(m.messages)-2]
		}
		m.lastInteractionFailed = false
	}

	m.messages = append(m.messages, core.Message{Type: core.UserMessage, Content: input})
	m.textArea.Reset()
	m.textArea.SetHeight(1)

	return m.startGeneration()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type != tea.KeyCtrlC {
			m.ctrlCPressed = false
		}

		// Handle scrolling regardless of state.
		switch msg.Type {
		case tea.KeyCtrlU:
			m.viewport.HalfViewUp()
			return m, nil
		case tea.KeyCtrlD:
			m.viewport.HalfViewDown()
			return m, nil
		}

		switch m.state {
		case stateThinking, stateGenerating, stateCancelling:
			switch msg.Type {
			case tea.KeyCtrlC:
				if m.state != stateCancelling && m.cancelGeneration != nil {
					m.cancelGeneration()
					m.state = stateCancelling
				}
			}
			return m, nil

		case stateIdle:
			switch msg.Type {
			case tea.KeyCtrlC:
				if m.textArea.Value() != "" {
					m.textArea.Reset()
					m.ctrlCPressed = false
					return m, nil
				}
				if m.ctrlCPressed {
					m.quitting = true
					return m, tea.Quit
				}
				m.ctrlCPressed = true
				return m, ctrlCTimeout()

			case tea.KeyEscape:
				// Clears the text input. If the palette is open, this will also
				// cause it to close in the subsequent logic.
				if m.textArea.Value() != "" {
					m.textArea.Reset()
				}
				return m, nil

			case tea.KeyTab, tea.KeyShiftTab:
				totalItems := len(m.paletteFilteredActions) + len(m.paletteFilteredCommands)
				if m.showPalette && totalItems > 0 {
					if msg.Type == tea.KeyTab {
						m.paletteCursor = (m.paletteCursor + 1) % totalItems
					} else { // Shift+Tab
						m.paletteCursor--
						if m.paletteCursor < 0 {
							m.paletteCursor = totalItems - 1
						}
					}
				}
				return m, nil

			case tea.KeyEnter:
				// If palette is shown, Enter selects the item.
				if m.showPalette {
					numActions := len(m.paletteFilteredActions)
					totalItems := numActions + len(m.paletteFilteredCommands)
					if totalItems > 0 {
						var selectedItem string
						if m.paletteCursor < numActions {
							selectedItem = m.paletteFilteredActions[m.paletteCursor]
						} else {
							selectedItem = m.paletteFilteredCommands[m.paletteCursor-numActions]
						}
						m.textArea.SetValue(selectedItem + " ")
						m.textArea.CursorEnd()
						return m, nil
					}
				}

				// Smart enter: submit if it's a command.
				if strings.HasPrefix(m.textArea.Value(), ":") {
					return m.handleSubmit()
				}
				// Otherwise, fall through to let the textarea handle the newline.

			case tea.KeyCtrlJ:
				return m.handleSubmit()
			}
		}

	case spinner.TickMsg:
		// Tick the spinner during all generation phases.
		if m.state != stateThinking && m.state != stateGenerating && m.state != stateCancelling {
			return m, nil
		}

		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)

		// If we are in the "thinking" state, the spinner is in the viewport.
		// We need to update the viewport's content to reflect the spinner's animation.
		if m.state == stateThinking {
			m.viewport.SetContent(m.renderConversation())
		}
		return m, spinnerCmd

	case streamResultMsg:
		lastMsg := &m.messages[len(m.messages)-1]
		if m.state == stateThinking {
			m.state = stateGenerating
			lastMsg.Content += string(msg)
			return m, tea.Batch(listenForStream(m.streamSub), renderTick())
		}
		lastMsg.Content += string(msg)
		return m, listenForStream(m.streamSub)

	case streamFinishedMsg:
		m.isStreaming = false

		if m.state == stateCancelling {
			// This was a cancellation.
			m.messages[len(m.messages)-1].Content = "Generation cancelled."
			m.messages[len(m.messages)-1].Type = core.CommandResultMessage // Re-use style for notification
			m.lastInteractionFailed = true
		}

		wasAtBottom := m.viewport.AtBottom()
		m.viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.viewport.GotoBottom()
		}

		m.state = stateIdle
		m.streamSub = nil
		m.cancelGeneration = nil
		m.textArea.Reset()
		m.textArea.Focus()

		if m.lastInteractionFailed {
			return m, nil // Don't count tokens on failure/cancellation
		}

		prompt := core.BuildPrompt(m.systemInstructions, m.providedDocuments, m.projectSourceCode, m.messages)
		m.isCountingTokens = true

		return m, countTokensCmd(prompt)

	case renderTickMsg:
		if m.state != stateGenerating || !m.isStreaming {
			return m, nil
		}

		lastMsg := m.messages[len(m.messages)-1]
		if lastMsg.Content != m.lastRenderedAIPart {
			wasAtBottom := m.viewport.AtBottom()
			m.viewport.SetContent(m.renderConversation())
			if wasAtBottom {
				m.viewport.GotoBottom()
			}
			m.lastRenderedAIPart = lastMsg.Content
		}

		return m, renderTick()

	case tokenCountResultMsg:
		m.tokenCount = int(msg)
		m.isCountingTokens = false
		return m, nil

	case ctrlCTimeoutMsg:
		m.ctrlCPressed = false
		return m, nil

	case initialContextLoadedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Error loading initial context:**\n```\n%v\n```\n", msg.err)
			m.messages = append(m.messages, core.Message{Type: core.CommandErrorResultMessage, Content: errorContent})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil
		}

		m.systemInstructions = msg.systemInstructions
		m.providedDocuments = msg.providedDocuments
		m.projectSourceCode = msg.projectSourceCode

		// Now that context is loaded, count the tokens.
		initialPrompt := core.BuildPrompt(m.systemInstructions, m.providedDocuments, m.projectSourceCode, nil)
		m.isCountingTokens = true

		return m, countTokensCmd(initialPrompt)

	case errorMsg:
		m.isStreaming = false

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		m.messages[len(m.messages)-1].Content = errorContent
		m.lastInteractionFailed = true

		wasAtBottom := m.viewport.AtBottom()
		m.viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.viewport.GotoBottom()
		}
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

	// After textarea update, check for palette
	if m.state == stateIdle {
		val := m.textArea.Value()
		isPaletteTrigger := strings.HasPrefix(val, ":") && !strings.Contains(val, " ")

		if isPaletteTrigger {
			prefix := strings.TrimPrefix(val, ":")
			newPaletteActions := []string{}
			for _, a := range m.availableActions {
				if strings.HasPrefix(a, prefix) {
					newPaletteActions = append(newPaletteActions, ":"+a)
				}
			}
			m.paletteFilteredActions = newPaletteActions

			newPaletteCommands := []string{}
			for _, c := range m.availableCommands {
				if strings.HasPrefix(c, prefix) {
					newPaletteCommands = append(newPaletteCommands, ":"+c)
				}
			}
			m.paletteFilteredCommands = newPaletteCommands

			totalItems := len(m.paletteFilteredActions) + len(m.paletteFilteredCommands)
			m.showPalette = totalItems > 0

			if m.paletteCursor >= totalItems {
				m.paletteCursor = 0
			}
		} else {
			m.showPalette = false
			m.paletteCursor = 0
		}
	} else {
		m.showPalette = false
		m.paletteCursor = 0
	}

	helpViewHeight := lipgloss.Height(m.helpView())

	paletteHeight := 0
	if m.showPalette {
		// We need a view to calculate its height.
		// This is a bit inefficient but necessary with lipgloss.
		paletteHeight = lipgloss.Height(m.paletteView())
	}

	viewportHeight := m.height - m.textArea.Height() - helpViewHeight - paletteHeight - textAreaStyle.GetVerticalPadding() - 2
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Height = viewportHeight

	return m, tea.Batch(cmds...)
}
