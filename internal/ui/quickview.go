package ui

import (
	"coder/internal/types"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/rmhubbert/bubbletea-overlay"
)

// QuickViewModel is the model for the quick view overlay.
type QuickViewModel struct {
	Viewport        viewport.Model
	Visible         bool
	GlamourRenderer *glamour.TermRenderer
	messages        []types.Message
	needsRender     bool
}

// NewQuickView creates a new quick view model.
func NewQuickView() *QuickViewModel {
	vp := viewport.New(80, 20) // Initial size, will be updated
	return &QuickViewModel{
		Viewport:    vp,
		Visible:     false,
		needsRender: false,
	}
}

func (m *QuickViewModel) SetMessages(messages []types.Message) {
	m.messages = messages
	m.needsRender = true
}

func (m *QuickViewModel) Init() tea.Cmd {
	return nil
}

func (m *QuickViewModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return cmd
}

func (m *QuickViewModel) renderMessages() string {
	var parts []string
	for _, msg := range m.messages {
		var renderedMsg string
		switch msg.Type {
		case types.UserMessage:
			blockWidth := m.Viewport.Width - userInputStyle.GetHorizontalFrameSize()
			renderedMsg = userInputStyle.Width(blockWidth).Render(msg.Content)
		case types.AIMessage:
			if msg.Content == "" {
				continue
			}
			renderedAI, err := m.GlamourRenderer.Render(msg.Content)
			if err != nil {
				renderedAI = msg.Content
			}
			renderedMsg = renderedAI
		case types.CommandMessage:
			blockWidth := m.Viewport.Width - commandInputStyle.GetHorizontalFrameSize()
			renderedMsg = commandInputStyle.Width(blockWidth).Render(msg.Content)
		case types.ImageMessage:
			blockWidth := m.Viewport.Width - imageMessageStyle.GetHorizontalFrameSize()
			renderedMsg = imageMessageStyle.Width(blockWidth).Render("Image: " + msg.Content)
		case types.CommandResultMessage:
			blockWidth := m.Viewport.Width - commandResultStyle.GetHorizontalFrameSize()
			renderedMsg = commandResultStyle.Width(blockWidth).Render(msg.Content)
		case types.CommandErrorResultMessage:
			blockWidth := m.Viewport.Width - commandErrorStyle.GetHorizontalFrameSize()
			renderedMsg = commandErrorStyle.Width(blockWidth).Render(msg.Content)
		default:
			continue
		}
		parts = append(parts, renderedMsg)
	}
	return strings.Join(parts, "\n")
}

func (m *QuickViewModel) View() string {
	if !m.Visible {
		return ""
	}
	return paletteContainerStyle.Render(m.Viewport.View())
}

// QuickViewOverlay implements the Overlay interface for the quick view.
type QuickViewOverlay struct{}

func (f *QuickViewOverlay) IsVisible(main *Model) bool {
	return main.QuickView.Visible
}

func (f *QuickViewOverlay) View(main *Model) string {
	quickViewWidth := main.Width * 3 / 4
	quickViewHeight := main.Height * 3 / 4

	main.QuickView.Viewport.Width = quickViewWidth - paletteContainerStyle.GetHorizontalFrameSize()
	main.QuickView.Viewport.Height = quickViewHeight - paletteContainerStyle.GetVerticalFrameSize()
	main.QuickView.GlamourRenderer = main.GlamourRenderer

	if main.QuickView.needsRender {
		main.QuickView.Viewport.SetContent(main.QuickView.renderMessages())
		main.QuickView.Viewport.GotoBottom()
		main.QuickView.needsRender = false
	}

	quickViewContent := main.QuickView.View()
	if quickViewContent == "" {
		return main.View()
	}

	quickViewModel := simpleModel{content: quickViewContent}

	overlayModel := overlay.New(
		quickViewModel,
		main,
		overlay.Center,
		overlay.Center,
		0,
		0,
	)
	return overlayModel.View()
}
