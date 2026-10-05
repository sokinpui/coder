package ui

import (
	"github.com/charmbracelet/lipgloss"
)

type ConfirmOverlay struct{}

func (c *ConfirmOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayConfirm && main.ConfirmRequest != nil
}

func (c *ConfirmOverlay) View(main *Model) string {
	if main.ConfirmRequest == nil {
		return main.View()
	}

	req := main.ConfirmRequest
	modalWidth := min(84, max(50, main.Width-4))
	contentWidth := max(20, modalWidth-6)

	header := PaletteHeaderStyle.Render("── Tool Execution Approval ──")
	toolName := ToolCallStyle.Render("⚡ " + req.ToolName)

	renderer := DefaultToolRegistry.Get(req.ToolName)
	details := renderer.RenderConfirmPrompt(req.Arguments, contentWidth)

	actions := lipgloss.JoinHorizontal(
		lipgloss.Top,
		ToolSuccessStyle.Bold(true).Render("[y] Allow"),
		"    ",
		ModelInfoStyle.Bold(true).Render("[a] Always Allow (session)"),
		"    ",
		ToolErrorStyle.Bold(true).Render("[n] Deny"),
	)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		toolName,
		"\n",
		details,
		"\n",
		actions,
	)

	content := PaletteContainerStyle.Width(modalWidth).Render(body)
	return OverlayCenter(content, main.View())
}
