package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/pkg/version"
)

func (m Model) StatusView() string {
	var title string
	if m.Chat.AnimatingTitle {
		title = m.Chat.DisplayedTitle
	} else {
		title = m.Session.GetTitle()
	}
	titlePart := StatusBarTitleStyle.MaxWidth(m.Width).Render(title)

	if m.Chat.CtrlCPressed && m.State == stateIdle {
		return RenderStatusBar(m.Width, titlePart, []string{StatusStyle.Render("Press Ctrl+C again to quit.")})
	}

	var items []string

	if m.StatusBarMessage != "" {
		items = append(items, StatusBarMsgStyle.Render(m.StatusBarMessage))
	}

	items = append(items, ModelInfoStyle.Render(version.Get()))

	if m.TokenCount > 0 {
		items = append(items, TokenCountStyle.Render(fmt.Sprintf("Tokens: ≈%d", m.TokenCount)))
	}

	modelCode := m.Session.GetConfig().Coder.ModelCode
	if m.Session.Capabilities().Has(engine.CapToolLoop) {
		modelCode = m.Session.GetConfig().Agent.ModelCode
	}
	items = append(items, ModelInfoStyle.Render(fmt.Sprintf("Model: %s", modelCode)))

	if m.Session.Capabilities().Has(engine.CapToolToggle) {
		toolsStatus := "Tools: [Compact] (Ctrl+T)"
		if m.ToolsExpanded {
			toolsStatus = "Tools: [Expanded] (Ctrl+T)"
		}
		items = append(items, ToolMutedStyle.Render(toolsStatus))
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
		items = append(items, spinnerWithText)
	}

	if m.Chat.IsFetchingModels {
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, StatusStyle.Render("Fetching models "), m.Chat.Spinner.View())
		items = append(items, spinnerWithText)
	}

	if m.ActiveOverlay == overlayConfirm {
		items = append(items, StatusStyle.Render("-- CONFIRM TOOL --"))
	} else if m.ActiveOverlay == overlaySelector && m.Selector.Title != "" && !m.Selector.ShowSearch {
		items = append(items, StatusStyle.Render("-- ATOMIC MSG --"))
	}

	return RenderStatusBar(m.Width, titlePart, items)
}

func RenderStatusBar(width int, titleLine string, items []string) string {
	var filtered []string
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			filtered = append(filtered, item)
		}
	}
	statusLine := strings.Join(filtered, " | ")

	if titleLine != "" {
		titleLine = " " + titleLine
	}
	if statusLine != "" {
		statusLine = " " + statusLine
	}

	if width > 0 {
		if titleLine != "" {
			titleLine = ansi.Truncate(titleLine, width, "")
		}
		statusLine = ansi.Truncate(statusLine, width, "")
	}

	if titleLine == "" {
		return statusLine
	}
	return lipgloss.JoinVertical(lipgloss.Left, titleLine, statusLine)
}
