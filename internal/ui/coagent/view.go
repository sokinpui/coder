package coagentui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/ui/core"
	"github.com/sokinpui/coder/pkg/version"
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	b.WriteString(m.statusView())
	content := b.String()

	if m.showSelector {
		modalWidth := min(90, max(50, m.width-4))
		modalHeight := min(30, max(12, m.height-4))
		m.selector.Width = modalWidth
		m.selector.Height = modalHeight
		m.selector.SearchInput.Width = modalWidth - 20
		selectorContent := m.selector.View(nil)
		return core.OverlayCenter(selectorContent, content)
	}

	return b.String()
}

func (m Model) statusView() string {
	if m.statusBarMessage != "" {
		return core.StatusBarMsgStyle.Render(m.statusBarMessage)
	}

	if m.ctrlCPressed && m.state == stateInput && m.input.Model.Value() == "" {
		return core.StatusStyle.Render("Press Ctrl+C again to quit.\n")
	}

	titleText := m.session.Title
	if m.animatingTitle {
		titleText = m.displayTitle
	}
	titlePart := core.StatusBarTitleStyle.MaxWidth(m.width).Render(titleText)

	leftStatus := ""
	if m.activeOverlay == overlaySelector && m.selector.Title != "" && !m.selector.ShowSearch {
		leftStatus = core.StatusStyle.Render("-- ATOMIC MSG --")
	}

	var rightItems []string
	modelInfo := fmt.Sprintf("Model: %s", m.cfg.Agent.ModelCode)
	versionPart := core.ModelInfoStyle.Render(fmt.Sprintf("%s", version.Get()))
	modelPart := core.ModelInfoStyle.Render(modelInfo)

	if m.activeOverlay != overlaySelector || m.selector.ShowSearch {
		if m.tokenCount > 0 {
			tokenPart := core.TokenCountStyle.Render(fmt.Sprintf("Tokens: ≈%d", m.tokenCount))
			rightItems = append(rightItems, tokenPart)
		}
		rightItems = append(rightItems, versionPart, modelPart)

		toolsStatus := "Tools: [Compact] (Ctrl+T)"
		if m.toolsExpanded {
			toolsStatus = "Tools: [Expanded] (Ctrl+T)"
		}
		rightItems = append(rightItems, core.ToolMutedStyle.Render(toolsStatus))
	}

	switch m.state {
	case stateThinking, stateGenerating:
		statusStyle := core.ThinkingStatusStyle
		if m.state == stateGenerating {
			statusStyle = core.GeneratingStatusStyle
		}
		statusText := m.statusText
		if statusText == "" {
			statusText = "Thinking"
		}
		elapsed := time.Since(m.stateStart).Seconds()
		timerText := fmt.Sprintf("%s (%.1fs) ", statusText, elapsed)
		spinnerWithText := lipgloss.JoinHorizontal(lipgloss.Bottom, statusStyle.Render(timerText), m.spinner.View())
		if leftStatus == "" {
			leftStatus = spinnerWithText
		} else {
			rightItems = append([]string{spinnerWithText}, rightItems...)
		}
	}

	return core.RenderStatusBar(m.width, titlePart, leftStatus, rightItems)
}
