package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/pkg/version"
)

func (m Model) StatusView() string {
	// Line 1: Title
	var title string
	if m.Chat.AnimatingTitle {
		title = m.Chat.DisplayedTitle
	} else {
		title = m.Session.GetTitle()
	}
	titlePart := StatusBarTitleStyle.MaxWidth(m.Width).Render(title)

	// Line 2: Status
	var rightStatusItems []string
	var leftStatus string

	if m.StatusBarMessage != "" {
		leftStatus = StatusBarMsgStyle.Render(m.StatusBarMessage)
	} else if m.Chat.CtrlCPressed && m.State == stateIdle {
		leftStatus = StatusStyle.Render("Press Ctrl+C again to quit.")
	} else if m.ActiveOverlay == overlaySelector && m.Selector.Title != "" && !m.Selector.ShowSearch {
		leftStatus = StatusStyle.Render("-- ATOMIC MSG --")
	} else if m.ActiveOverlay == overlayConfirm {
		leftStatus = StatusStyle.Render("-- CONFIRM TOOL --")
	}

	hideRightInfo := leftStatus != ""

	modelCode := m.Session.GetConfig().Coder.ModelCode
	if m.Session.Capabilities().Has(engine.CapToolLoop) {
		modelCode = m.Session.GetConfig().Agent.ModelCode
	}
	modelInfo := fmt.Sprintf("Model: %s", modelCode)
	versionPart := ModelInfoStyle.Render(fmt.Sprintf("%s", version.Get()))

	modelPart := ModelInfoStyle.Render(modelInfo)

	if !hideRightInfo {
		if m.TokenCount > 0 {
			tokenPart := TokenCountStyle.Render(fmt.Sprintf("Tokens: ≈%d", m.TokenCount))
			rightStatusItems = append(rightStatusItems, tokenPart)
		}
		rightStatusItems = append(rightStatusItems, versionPart, modelPart)
	}

	if !hideRightInfo && m.Session.Capabilities().Has(engine.CapToolToggle) {
		toolsStatus := "Tools: [Compact] (Ctrl+T)"
		if m.ToolsExpanded {
			toolsStatus = "Tools: [Expanded] (Ctrl+T)"
		}
		rightStatusItems = append(rightStatusItems, ToolMutedStyle.Render(toolsStatus))
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
			statusStyle = AskingStatusStyle
		case stateThinking:
			statusText = "Thinking"
			if m.StatusText != "" {
				statusText = m.StatusText
			}
			statusStyle = ThinkingStatusStyle
		case stateGenerating:
			statusText = "Generating"
			statusStyle = GeneratingStatusStyle
		}

		elapsed := time.Since(m.Chat.StateStartTime).Seconds()
		timerText := fmt.Sprintf("%s (%.1fs) ", statusText, elapsed)
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, statusStyle.Render(timerText), m.Chat.Spinner.View())
		rightStatusItems = append(rightStatusItems, spinnerWithText)
	}
	if m.Chat.IsFetchingModels {
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, StatusStyle.Render("Fetching models "), m.Chat.Spinner.View())
		rightStatusItems = append(rightStatusItems, spinnerWithText)
	}

	return RenderStatusBar(m.Width, titlePart, leftStatus, rightStatusItems)
}
