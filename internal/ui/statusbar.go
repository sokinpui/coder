package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/utils"
)

func (m Model) StatusView() string {
	if m.StatusBarMessage != "" {
		return statusBarMsgStyle.Render(m.StatusBarMessage)
	}

	if m.Chat.CtrlCPressed && m.State == stateIdle && m.Chat.TextArea.Value() == "" {
		return statusStyle.Render("Press Ctrl+C again to quit.\n")
	}

	if m.State == stateHistorySelect {
		return statusStyle.Render("j/k: move | gg/G: top/bottom | enter: load | esc: cancel")
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

	switch m.State {
	case stateAtomicMsg:
		helpStr := "j/k: move | v: select | y: yank | d: del | a: apply | e: edit | r: regen | b: branch | esc/C-c: exit"
		leftStatus = statusStyle.Render(fmt.Sprintf("-- ATOMIC MSG -- | %s", helpStr))
	}

	modelInfo := fmt.Sprintf("Model: %s", m.Session.GetConfig().Generation.ModelCode)
	versionPart := modelInfoStyle.Render(fmt.Sprintf("%s", utils.GetVersion()))

	modelPart := modelInfoStyle.Render(modelInfo)

	if m.State != stateAtomicMsg {
		if m.TokenCount > 0 {
			tokenPart := tokenCountStyle.Render(fmt.Sprintf("Tokens: ≈%d", m.TokenCount))
			rightStatusItems = append(rightStatusItems, tokenPart)
		}
		rightStatusItems = append(rightStatusItems, versionPart, modelPart)
	}

	switch m.State {
	case stateAsking, stateThinking, stateGenerating, stateCancelling:
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
			statusStyle = thinkingStatusStyle
		case stateGenerating:
			statusText = "Generating"
			statusStyle = generatingStatusStyle
		case stateCancelling:
			statusText = "Cancelling"
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

	var filteredStatusItems []string
	for _, item := range rightStatusItems {
		if strings.TrimSpace(item) != "" {
			filteredStatusItems = append(filteredStatusItems, item)
		}
	}
	rightStatus := strings.Join(filteredStatusItems, " | ")

	var statusLine string
	if leftStatus != "" {
		spacing := max(m.Width-lipgloss.Width(leftStatus)-lipgloss.Width(rightStatus), 1)
		statusLine = lipgloss.JoinHorizontal(lipgloss.Top, leftStatus, strings.Repeat(" ", spacing), rightStatus)
	} else {
		statusLine = rightStatus
	}

	return lipgloss.JoinVertical(lipgloss.Left, titlePart, statusLine)
}
