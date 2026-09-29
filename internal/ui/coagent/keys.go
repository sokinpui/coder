package coagentui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/core"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type != tea.KeyCtrlC {
		m.ctrlCPressed = false
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		if m.state != stateInput {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
			m.isAIRendering = false
			m.pendingAIRender = false
			m.state = stateInput
			m.input.Model.Focus()
			m.session.Messages = append(m.session.Messages, types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: "[Interrupted by user]",
			})
			_ = m.session.SaveConversation()
			m = m.updateLayout()
			return m.updateViewportContent(), nil
		}
		if m.input.Model.Value() != "" {
			m.input.Model.Reset()
			m.ctrlCPressed = false
			m = m.updateLayout()
			return m.updateViewportContent(), nil
		}
		if m.ctrlCPressed {
			return m, tea.Quit
		}
		m.ctrlCPressed = true
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return ctrlCTimeoutMsg{}
		})

	case tea.KeyEscape:
		if m.state == stateInput {
			if m.input.Model.Value() != "" {
				m.input.Model.Reset()
				m.ctrlCPressed = false
				m = m.updateLayout()
				return m.updateViewportContent(), nil
			}
			return m.openAtomicMsgMode()
		}

	case tea.KeyCtrlD, tea.KeyPgDown:
		m.viewport.HalfPageDown()
		return m, nil

	case tea.KeyCtrlU, tea.KeyPgUp:
		m.viewport.HalfPageUp()
		return m, nil

	case tea.KeyCtrlJ:
		if m.state == stateInput {
			text := strings.TrimSpace(m.input.Model.Value())
			if text == "/msg" || text == "/cards" {
				m.input.Model.Reset()
				m = m.updateLayout()
				return m.openAtomicMsgMode()
			}
			if text != "" {
				m.input.Model.Reset()
				m = m.updateLayout()
				return m.startPrompt(text)
			}
			return m, nil
		}

	case tea.KeyEnter:
		if m.state == stateInput {
			text := strings.TrimSpace(m.input.Model.Value())
			if text == "/exit" || text == "/quit" || text == "exit" || text == "quit" {
				return m, tea.Quit
			}
			if text == "/msg" || text == "/cards" {
				m.input.Model.Reset()
				m = m.updateLayout()
				return m.openAtomicMsgMode()
			}
		}

	case tea.KeyCtrlT:
		m.toolsExpanded = !m.toolsExpanded
		m.viewport.ToolsExpanded = m.toolsExpanded
		m.viewport.ToolExpandedCache = make(map[int][]string)
		m = m.updateViewportContent()
		return m, nil
	}

	km := m.cfg.Agent.Keymap

	switch msg.String() {
	case km.Editor, "ctrl+e":
		if m.state == stateInput {
			return m, editInEditorCmd(m.input.Model.Value())
		}
	case km.Submit, "ctrl+j":
		if m.state == stateInput {
			text := strings.TrimSpace(m.input.Model.Value())
			if text == "/msg" || text == "/cards" {
				m.input.Model.Reset()
				m = m.updateLayout()
				return m.openAtomicMsgMode()
			}
			if text != "" {
				m.input.Model.Reset()
				m = m.updateLayout()
				return m.startPrompt(text)
			}
			return m, nil
		}
	case km.Paste, "ctrl+v":
		if m.state == stateInput {
			return m, core.HandlePasteCmd(m.cfg)
		}
	case km.History, "ctrl+h":
		if m.state == stateInput {
			return m.openHistorySelector(0)
		}
	case km.New, "ctrl+n":
		if m.state == stateInput {
			return m.newSession()
		}
	case "ctrl+u":
		m.viewport.HalfPageUp()
		return m, nil
	case "ctrl+d":
		m.viewport.HalfPageDown()
		return m, nil
	case "esc":
		if m.state == stateInput {
			if m.input.Model.Value() != "" {
				m.input.Model.Reset()
				m.ctrlCPressed = false
				m = m.updateLayout()
				return m.updateViewportContent(), nil
			}
			return m.openAtomicMsgMode()
		}
	}

	if m.state != stateInput {
		switch msg.Type {
		case tea.KeyUp:
			m.viewport.LineUp(1)
		case tea.KeyDown:
			m.viewport.LineDown(1)
		}
		return m, nil
	}

	if m.state == stateInput {
		var cmd tea.Cmd
		m.input.Model, cmd = m.input.Model.Update(msg)
		m = m.updateLayout()
		return m, cmd
	}

	return m, nil
}
