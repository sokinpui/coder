package ui

import (
	"coder/internal/core"
	"coder/internal/session"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, loadInitialContextCmd(m.session))
}

func (m Model) newSession() (Model, tea.Cmd) {
	// The session handles saving and clearing messages.
	// The UI just needs to reset its state.
	m.session.AddMessage(core.Message{Type: core.InitMessage, Content: welcomeMessage})

	// Reset UI and state flags.
	m.lastInteractionFailed = false
	m.lastRenderedAIPart = ""
	m.textArea.Reset()
	m.textArea.SetHeight(1)
	m.textArea.Focus()
	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderConversation())

	// Recalculate the token count for the base context.
	m.isCountingTokens = true
	return m, countTokensCmd(m.session.GetInitialPromptForTokenCount())
}

func (m Model) handleSubmit() (tea.Model, tea.Cmd) {
	input := m.textArea.Value()

	// If the last interaction failed, the user is likely retrying.
	// Clear the previous failed attempt before submitting the new prompt.
	if !strings.HasPrefix(input, ":") && m.lastInteractionFailed {
		m.session.RemoveLastInteraction()
		m.lastInteractionFailed = false
	}

	event := m.session.HandleInput(input)

	switch event.Type {
	case session.NoOp:
		return m, nil

	case session.MessagesUpdated:
		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()
		m.textArea.Reset()
		m.textArea.SetHeight(1)
		return m, nil

	case session.NewSessionStarted:
		return m.newSession()

	case session.GenerationStarted:
		m.state = stateThinking
		m.isStreaming = true
		m.streamSub = event.Data.(chan string)
		m.textArea.Blur()
		m.textArea.Reset()
		m.textArea.SetHeight(1)

		m.lastRenderedAIPart = ""
		m.lastInteractionFailed = false // Reset this flag for the new generation attempt.

		m.viewport.SetContent(m.renderConversation())
		m.viewport.GotoBottom()

		prompt := m.session.GetPromptForTokenCount()
		m.isCountingTokens = true
		return m, tea.Batch(listenForStream(m.streamSub), m.spinner.Tick, countTokensCmd(prompt))
	}

	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Reset cycling flag on any key press that is not Tab.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type != tea.KeyTab && key.Type != tea.KeyShiftTab {
		m.isCyclingCompletions = false
	}

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
				if m.state != stateCancelling {
					m.session.CancelGeneration()
					m.state = stateCancelling
				}
			}
			return m, nil

		case stateIdle:
			switch msg.Type {
			case tea.KeyCtrlC:
				if m.textArea.Value() != "" {
					m.clearedInputBuffer = m.textArea.Value()
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

			case tea.KeyCtrlZ:
				if m.clearedInputBuffer != "" {
					m.textArea.SetValue(m.clearedInputBuffer)
					m.textArea.CursorEnd()
					m.clearedInputBuffer = ""
				}
				return m, nil

			case tea.KeyEscape:
				// Clears the text input. If the palette is open, this will also
				// cause it to close in the subsequent logic.
				if m.textArea.Value() != "" {
					m.textArea.Reset()
				}
				return m, nil

			case tea.KeyTab, tea.KeyShiftTab:
				m.isCyclingCompletions = true

				numActions := len(m.paletteFilteredActions)
				numCommands := len(m.paletteFilteredCommands)
				numArgs := len(m.paletteFilteredArguments)
				totalItems := numActions + numCommands + numArgs

				if !m.showPalette || totalItems == 0 {
					return m, nil
				}

				if msg.Type == tea.KeyTab {
					m.paletteCursor = (m.paletteCursor + 1) % totalItems
				} else { // Shift+Tab
					m.paletteCursor--
					if m.paletteCursor < 0 {
						m.paletteCursor = totalItems - 1
					}
				}

				var selectedItem string
				isArgument := false
				if m.paletteCursor < numActions {
					selectedItem = m.paletteFilteredActions[m.paletteCursor]
				} else if m.paletteCursor < numActions+numCommands {
					selectedItem = m.paletteFilteredCommands[m.paletteCursor-numActions]
				} else {
					selectedItem = m.paletteFilteredArguments[m.paletteCursor-numActions-numCommands]
					isArgument = true
				}

				val := m.textArea.Value()
				parts := strings.Fields(val)

				if isArgument {
					var prefixParts []string
					if len(parts) > 0 && !strings.HasSuffix(val, " ") {
						prefixParts = parts[:len(parts)-1]
					} else {
						prefixParts = parts
					}
					m.textArea.SetValue(strings.Join(append(prefixParts, selectedItem), " "))
				} else { // command/action
					m.textArea.SetValue(selectedItem)
				}
				m.textArea.CursorEnd()
				return m, nil

			case tea.KeyEnter:
				// Smart enter: submit if it's a command.
				if strings.HasPrefix(m.textArea.Value(), ":") {
					return m.handleSubmit()
				}
				// Otherwise, fall through to let the textarea handle the newline.
				break

			case tea.KeyCtrlE:
				if m.textArea.Focused() {
					return m, editInEditorCmd(m.textArea.Value())
				}

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
		messages := m.session.GetMessages()
		lastMsg := messages[len(messages)-1]
		if m.state == stateThinking {
			m.state = stateGenerating
			lastMsg.Content += string(msg)
			m.session.ReplaceLastMessage(lastMsg)
			return m, tea.Batch(listenForStream(m.streamSub), renderTick())
		}
		lastMsg.Content += string(msg)
		m.session.ReplaceLastMessage(lastMsg)
		return m, listenForStream(m.streamSub)

	case streamFinishedMsg:
		m.isStreaming = false
		messages := m.session.GetMessages()

		if m.state == stateCancelling {
			// This was a cancellation.
			lastMsg := messages[len(messages)-1]
			lastMsg.Content = "Generation cancelled."
			lastMsg.Type = core.CommandResultMessage // Re-use style for notification
			m.session.ReplaceLastMessage(lastMsg)
			m.lastInteractionFailed = true
		}

		wasAtBottom := m.viewport.AtBottom()
		m.viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.viewport.GotoBottom()
		}

		m.state = stateIdle
		m.streamSub = nil
		m.session.CancelGeneration()
		m.textArea.Reset()
		m.textArea.Focus()

		if m.lastInteractionFailed {
			return m, nil // Don't count tokens on failure/cancellation
		}

		prompt := m.session.GetPromptForTokenCount()
		m.isCountingTokens = true

		return m, countTokensCmd(prompt)

	case editorFinishedMsg:
		if msg.err != nil {
			errorContent := fmt.Sprintf("\n**Editor Error:**\n```\n%v\n```\n", msg.err)
			m.session.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: errorContent})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil
		}
		m.textArea.SetValue(msg.content)
		m.textArea.CursorEnd()
		m.textArea.Focus()
		return m, nil

	case renderTickMsg:
		if m.state != stateGenerating || !m.isStreaming {
			return m, nil
		}

		messages := m.session.GetMessages()
		lastMsg := messages[len(messages)-1]
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
			m.session.AddMessage(core.Message{Type: core.CommandErrorResultMessage, Content: errorContent})
			m.viewport.SetContent(m.renderConversation())
			m.viewport.GotoBottom()
			return m, nil
		}

		// Now that context is loaded, count the tokens.
		m.isCountingTokens = true
		return m, countTokensCmd(m.session.GetInitialPromptForTokenCount())

	case errorMsg:
		m.isStreaming = false

		errorContent := fmt.Sprintf("\n**Error:**\n```\n%v\n```\n", msg.error)
		messages := m.session.GetMessages()
		lastMsg := messages[len(messages)-1]
		lastMsg.Content = errorContent
		m.session.ReplaceLastMessage(lastMsg)
		m.lastInteractionFailed = true

		wasAtBottom := m.viewport.AtBottom()
		m.viewport.SetContent(m.renderConversation())
		if wasAtBottom {
			m.viewport.GotoBottom()
		}
		m.state = stateIdle
		m.streamSub = nil
		m.session.CancelGeneration()
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
	if !m.isCyclingCompletions {
		val := m.textArea.Value()
		m.paletteFilteredActions = []string{}
		m.paletteFilteredCommands = []string{}
		m.paletteFilteredArguments = []string{}

		if m.state == stateIdle && strings.HasPrefix(val, ":") {
			parts := strings.Fields(val)
			hasTrailingSpace := strings.HasSuffix(val, " ")

			if len(parts) == 0 { // Just ":"
				parts = []string{":"}
			}

			if len(parts) == 1 && !hasTrailingSpace {
				// Command/Action completion mode
				prefix := strings.TrimPrefix(parts[0], ":")
				for _, a := range m.availableActions {
					if strings.HasPrefix(a, prefix) {
						m.paletteFilteredActions = append(m.paletteFilteredActions, ":"+a)
					}
				}
				for _, c := range m.availableCommands {
					if strings.HasPrefix(c, prefix) {
						m.paletteFilteredCommands = append(m.paletteFilteredCommands, ":"+c)
					}
				}
			} else if len(parts) >= 1 {
				// Argument completion mode
				cmdName := strings.TrimPrefix(parts[0], ":")
				suggestions := core.GetCommandArgumentSuggestions(cmdName, m.session.GetConfig())
				if suggestions != nil {
					var argPrefix string
					if len(parts) > 1 && !hasTrailingSpace {
						argPrefix = parts[len(parts)-1]
					}

					for _, s := range suggestions {
						if strings.HasPrefix(s, argPrefix) {
							m.paletteFilteredArguments = append(m.paletteFilteredArguments, s)
						}
					}
				}
			}
		}

		totalItems := len(m.paletteFilteredActions) + len(m.paletteFilteredCommands) + len(m.paletteFilteredArguments)
		m.showPalette = totalItems > 0

		if m.paletteCursor >= totalItems {
			m.paletteCursor = 0
		}
	}

	statusViewHeight := lipgloss.Height(m.statusView())

	paletteHeight := 0
	if m.showPalette {
		// We need a view to calculate its height.
		// This is a bit inefficient but necessary with lipgloss.
		paletteHeight = lipgloss.Height(m.paletteView())
	}

	viewportHeight := m.height - m.textArea.Height() - statusViewHeight - paletteHeight - textAreaStyle.GetVerticalPadding() - 2
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	m.viewport.Height = viewportHeight

	return m, tea.Batch(cmds...)
}
