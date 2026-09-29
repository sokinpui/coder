package coagentui

import (
	"fmt"
	"strings"
	"time"

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
	return b.String()
}

func (m Model) statusView() string {
	if m.ctrlCPressed && m.state == stateInput && m.input.Model.Value() == "" {
		return core.StatusStyle.Render("Press Ctrl+C again to quit.\n")
	}

	title := core.StatusBarTitleStyle.Render("Co Autonomous Agent")
	leftStatus := ""
	var rightItems []string
	if m.tokenCount > 0 {
		rightItems = append(rightItems, core.TokenCountStyle.Render(fmt.Sprintf("Tokens: ≈%d", m.tokenCount)))
	}
	rightItems = append(rightItems, core.ModelInfoStyle.Render(fmt.Sprintf("%s", version.Get())))
	rightItems = append(rightItems, core.ModelInfoStyle.Render(fmt.Sprintf("Model: %s", m.cfg.Generation.ModelCode)))

	if m.state != stateInput {
		elapsed := time.Since(m.stateStart).Seconds()
		timerText := fmt.Sprintf("%s (%.1fs) %s", m.statusText, elapsed, m.spinner.View())
		rightItems = append(rightItems, core.GeneratingStatusStyle.Render(timerText))
	}

	return core.RenderStatusBar(m.width, title, leftStatus, rightItems)
}
