package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) StatusView() string {
	if m.StatusBarMessage != "" {
		return statusBarMsgStyle.Render(m.StatusBarMessage)
	}

	if m.CtrlCPressed && m.State == stateIdle && m.TextArea.Value() == "" {
		return statusStyle.Render("Press Ctrl+C again to quit.")
	}

	if m.State == stateHistorySelect {
		return statusStyle.Render("j/k: move | gg/G: top/bottom | enter: load | esc: cancel")
	}

	if m.State == stateSearching {
		searchStatus := "enter: confirm | esc: cancel"
		if len(m.SearchResultLines) > 0 {
			searchStatus = fmt.Sprintf("match %d of %d | %s", m.CurrentSearchResult+1, len(m.SearchResultLines), searchStatus)
		} else if m.SearchQuery != "" {
			searchStatus = fmt.Sprintf("no matches | %s", searchStatus)
		}
		return statusStyle.Render(searchStatus)
	}

	if m.State == stateSearchNav {
		return statusStyle.Render(fmt.Sprintf("Search: '%s' | n/N: navigate | /: new search | esc: exit", m.SearchQuery))
	}

	// Line 1: Title
	var title string
	if m.AnimatingTitle {
		title = m.DisplayedTitle
	} else {
		title = m.Session.GetTitle()
	}
	titlePart := statusBarTitleStyle.MaxWidth(m.Width).Render(title)

	// Line 2: Status
	var leftStatus string
	switch m.State {
	case stateVisualSelect:
		var modeStr, helpStr string
		switch m.VisualMode {
		case visualModeGenerate:
			modeStr = "GENERATE"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		case visualModeEdit:
			modeStr = "EDIT"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		case visualModeBranch:
			modeStr = "BRANCH"
			helpStr = "j/k: move | enter: confirm | esc: cancel"
		default: // visualModeNone
			modeStr = "VISUAL"
			if m.VisualIsSelecting {
				helpStr = "j/k: move | o/O: swap cursor | y: copy | d: delete | esc: cancel selection"
			} else {
				helpStr = "j/k: move | v: start selection | esc: cancel"
			}
		}
		leftStatus = statusStyle.Render(fmt.Sprintf("-- %s MODE -- | %s", modeStr, helpStr))
	case stateCancelling:
		leftStatus = generatingStatusStyle.Render("Cancelling...")
	}

	modeInfo := fmt.Sprintf("Mode: %s", m.Session.GetConfig().AppMode)
	modelInfo := fmt.Sprintf("Model: %s", m.Session.GetConfig().Generation.ModelCode)
	tempInfo := fmt.Sprintf("Temp: %.1f", m.Session.GetConfig().Generation.Temperature)

	var tokenInfo string
	if m.IsCountingTokens {
		tokenInfo = "Tokens: counting..."
	} else if m.TokenCount > 0 {
		tokenInfo = fmt.Sprintf("Tokens: %d", m.TokenCount)
	}

	modePart := modelInfoStyle.Render(modeInfo)
	modelPart := modelInfoStyle.Render(modelInfo)
	tempPart := modelInfoStyle.Render(tempInfo)
	tokenPart := tokenCountStyle.Render(tokenInfo)

	rightStatusItems := []string{}
	if m.State != stateVisualSelect {
		if tokenPart != "" {
			rightStatusItems = append(rightStatusItems, tokenPart)
		}
		rightStatusItems = append(rightStatusItems, modePart, modelPart, tempPart)
	}

	if m.State == stateGenPending || m.State == stateThinking || m.State == stateGenerating {
		statusText := "Thinking" // Default for genpending and thinking
		if m.State == stateGenerating {
			statusText = "Generating"
		}
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, statusStyle.Render(statusText+" "), m.Spinner.View())
		rightStatusItems = append(rightStatusItems, spinnerWithText)
	}
	rightStatus := strings.Join(rightStatusItems, " | ")

	var statusLine string
	if leftStatus != "" {
		spacing := m.Width - lipgloss.Width(leftStatus) - lipgloss.Width(rightStatus)
		if spacing < 1 {
			spacing = 1
		}
		statusLine = lipgloss.JoinHorizontal(lipgloss.Top, leftStatus, strings.Repeat(" ", spacing), rightStatus)
	} else {
		statusLine = rightStatus
	}

	return lipgloss.JoinVertical(lipgloss.Left, titlePart, statusLine)
}
