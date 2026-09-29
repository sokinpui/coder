package coderui

import (
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/core"
	"github.com/sokinpui/coder/internal/ui/core/markdown"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type QuickViewModel struct {
	Viewport        viewport.Model
	GlamourRenderer *markdown.Renderer
	messages        []types.Message
	needsRender     bool
}

func NewQuickView() *QuickViewModel {
	vp := viewport.New(80, 20) // Initial size, will be updated
	return &QuickViewModel{
		Viewport:    vp,
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

func (m *QuickViewModel) renderContent() string {
	var parts []string
	for _, msg := range m.messages {
		rendered := core.RenderMessage(msg, m.Viewport.Width, m.GlamourRenderer)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}
	return strings.Join(parts, "\n")
}

func (m *QuickViewModel) View() string {
	return paletteContainerStyle.Render(m.Viewport.View())
}

type QuickViewOverlay struct{}

func (f *QuickViewOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayQuickView
}

func (f *QuickViewOverlay) View(main *Model) string {
	quickViewWidth := main.Width * 3 / 4
	quickViewHeight := main.Height * 3 / 4

	main.QuickView.Viewport.Width = quickViewWidth - paletteContainerStyle.GetHorizontalFrameSize()
	main.QuickView.Viewport.Height = quickViewHeight - paletteContainerStyle.GetVerticalFrameSize()
	main.QuickView.GlamourRenderer = main.GlamourRenderer

	if main.QuickView.needsRender {
		main.QuickView.Viewport.SetContent(main.QuickView.renderContent())
		main.QuickView.Viewport.GotoTop()
		main.QuickView.needsRender = false
	}

	quickViewContent := main.QuickView.View()
	if quickViewContent == "" {
		return main.View()
	}

	return OverlayCenter(quickViewContent, main.View())
}
