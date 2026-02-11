package ui

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rmhubbert/bubbletea-overlay"
)

type jumpItem struct {
	msgIndex int
	content  string
}

type JumpModel struct {
	Items   []jumpItem
	Cursor  int
	Width   int
	Height  int
	Visible bool
}

func NewJump() JumpModel {
	return JumpModel{
		Visible: false,
	}
}

func (m *JumpModel) UpdateItems(messages []types.Message) {
	m.Items = []jumpItem{}
	for i, msg := range messages {
		if msg.Type == types.UserMessage {
			// Clean up content for preview: take first line and trim
			preview := strings.Split(msg.Content, "\n")[0]
			preview = strings.TrimSpace(preview)
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			m.Items = append(m.Items, jumpItem{
				msgIndex: i,
				content:  preview,
			})
		}
	}
	m.Cursor = len(m.Items) - 1
	if m.Cursor < 0 {
		m.Cursor = 0
	}
}

func (m JumpModel) Update(msg tea.Msg) (JumpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			m.Visible = false
			return m, nil

		case tea.KeyCtrlU:
			m.Cursor -= m.Height / 2
			if m.Cursor < 0 {
				m.Cursor = 0
			}
			return m, nil

		case tea.KeyCtrlD:
			m.Cursor += m.Height / 2
			if m.Cursor >= len(m.Items) {
				m.Cursor = len(m.Items) - 1
			}
			return m, nil

		case tea.KeyUp, tea.KeyCtrlK:
			if m.Cursor > 0 {
				m.Cursor--
			}
			return m, nil

		case tea.KeyDown, tea.KeyCtrlJ:
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if len(m.Items) > 0 && m.Cursor < len(m.Items) {
				selected := m.Items[m.Cursor]
				m.Visible = false
				return m, func() tea.Msg { return jumpResultMsg{msgIndex: selected.msgIndex} }
			}
			return m, nil
		}
	}

	return m, nil
}

func (m JumpModel) View() string {
	if !m.Visible {
		return ""
	}

	var b strings.Builder
	b.WriteString("Jump to message:\n\n")

	if len(m.Items) == 0 {
		b.WriteString("  No user messages found.")
		return paletteContainerStyle.Width(m.Width).Render(b.String())
	}

	maxItems := m.Height - 4
	if maxItems < 1 {
		maxItems = 5
	}

	start := 0
	if m.Cursor >= maxItems {
		start = m.Cursor - maxItems + 1
	}
	end := start + maxItems
	if end > len(m.Items) {
		end = len(m.Items)
	}

	for i, item := range m.Items[start:end] {
		actualIndex := i + start
		line := fmt.Sprintf("Msg %d: %s", item.msgIndex, item.content)
		if actualIndex == m.Cursor {
			b.WriteString(paletteSelectedItemStyle.Render("▸ " + line))
		} else {
			b.WriteString(paletteItemStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	return paletteContainerStyle.Width(m.Width).Render(b.String())
}

type JumpOverlay struct{}

func (f *JumpOverlay) IsVisible(main *Model) bool {
	return main.State == stateJump
}

func (f *JumpOverlay) View(main *Model) string {
	jumpWidth := main.Width / 2
	jumpHeight := main.Height / 2
	if jumpWidth < 60 {
		jumpWidth = 60
	}
	if jumpHeight < 10 {
		jumpHeight = 10
	}
	main.Jump.Width = jumpWidth
	main.Jump.Height = jumpHeight

	jumpContent := main.Jump.View()
	if jumpContent == "" {
		return main.View()
	}

	jumpModel := simpleModel{content: jumpContent}

	overlayModel := overlay.New(
		jumpModel,
		main,
		overlay.Center,
		overlay.Center,
		0,
		0,
	)
	return overlayModel.View()
}
