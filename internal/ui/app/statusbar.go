package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/ui/core"
	"github.com/sokinpui/coder/pkg/version"
)

func (m Model) StatusView() string {
	if m.StatusBarMessage != "" {
		return statusBarMsgStyle.Render(m.StatusBarMessage)
	}

	if m.Chat.CtrlCPressed && m.State == stateIdle && m.Chat.TextArea.Value() == "" {
		return statusStyle.Render("Press Ctrl+C again to quit.\n")
	}

	// Line 1: Title
	var title string
	if m.Chat.AnimatingTitle {
		title = m.Chat.DisplayedTitle
	} else {
		title = m.Session.GetTitle()
	}
	titlePart := statusBarTitleStyle.MaxWidth(m.Width).Render(title)

	// Line 2: Status
	var rightStatusItems []string
	var leftStatus string

	if m.ActiveOverlay == overlaySelector && m.Selector.Title != "" && !m.Selector.ShowSearch {
		leftStatus = statusStyle.Render("-- ATOMIC MSG --")
	}

	modelCode := m.Session.GetConfig().Coder.ModelCode
	if m.Session.Capabilities().Has(engine.CapToolLoop) {
		modelCode = m.Session.GetConfig().Agent.ModelCode
	}
	modelInfo := fmt.Sprintf("Model: %s", modelCode)
	versionPart := modelInfoStyle.Render(fmt.Sprintf("%s", version.Get()))

	modelPart := modelInfoStyle.Render(modelInfo)

	if m.ActiveOverlay != overlaySelector || m.Selector.ShowSearch {
		if m.TokenCount > 0 {
			tokenPart := tokenCountStyle.Render(fmt.Sprintf("Tokens: ≈%d", m.TokenCount))
			rightStatusItems = append(rightStatusItems, tokenPart)
		}
		rightStatusItems = append(rightStatusItems, versionPart, modelPart)
	}

	if m.Session.Capabilities().Has(engine.CapToolToggle) {
		toolsStatus := "Tools: [Compact] (Ctrl+T)"
		if m.ToolsExpanded {
			toolsStatus = "Tools: [Expanded] (Ctrl+T)"
		}
		rightStatusItems = append(rightStatusItems, core.ToolMutedStyle.Render(toolsStatus))
	}

	switch m.State {
	case stateAsking, stateThinking, stateGenerating:
		var (
			statusText  string
			statusStyle lipgloss.Style
		)
		switch m.State {
		case stateAsking:
			statusText = "Asking"
			statusStyle = askingStatusStyle
		case stateThinking:
			statusText = "Thinking"
			if m.StatusText != "" {
				statusText = m.StatusText
			}
			statusStyle = thinkingStatusStyle
		case stateGenerating:
			statusText = "Generating"
			statusStyle = generatingStatusStyle
		}

		elapsed := time.Since(m.Chat.StateStartTime).Seconds()
		timerText := fmt.Sprintf("%s (%.1fs) ", statusText, elapsed)
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, statusStyle.Render(timerText), m.Chat.Spinner.View())
		rightStatusItems = append(rightStatusItems, spinnerWithText)
	}
	if m.Chat.IsFetchingModels {
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, statusStyle.Render("Fetching models "), m.Chat.Spinner.View())
		rightStatusItems = append(rightStatusItems, spinnerWithText)
	}

	return core.RenderStatusBar(m.Width, titlePart, leftStatus, rightStatusItems)
}
