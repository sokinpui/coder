package ui

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/sokinpui/coder/internal/clipboard"
)

type SelectionPoint struct {
	Row int
	Col int
}

type TextSelection struct {
	Active   bool
	Dragging bool
	Start    SelectionPoint
	End      SelectionPoint
}

func (s TextSelection) IsEmpty() bool {
	return s.Start.Row == s.End.Row && s.Start.Col == s.End.Col
}

func (s TextSelection) Normalize() (SelectionPoint, SelectionPoint) {
	if s.Start.Row < s.End.Row {
		return s.Start, s.End
	}
	if s.Start.Row > s.End.Row {
		return s.End, s.Start
	}
	if s.Start.Col <= s.End.Col {
		return s.Start, s.End
	}
	return s.End, s.Start
}

func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.ActiveOverlay == overlayQuickView {
		if m.QuickView == nil {
			return m, nil
		}
		if msg.Button == tea.MouseButtonWheelUp && m.QuickView.Viewport.YOffset > 0 {
			m.QuickView.Viewport.LineUp(2)
			return m, nil
		}
		if msg.Button == tea.MouseButtonWheelDown && !m.QuickView.Viewport.AtBottom() {
			m.QuickView.Viewport.LineDown(2)
			return m, nil
		}
		return m, nil
	}

	if m.ActiveOverlay != overlayNone {
		return m, nil
	}

	if msg.Button == tea.MouseButtonWheelUp {
		if m.Chat.Viewport.YOffset <= 0 {
			return m, nil
		}
		m.Chat.AutoScroll = false
		m.Chat.Viewport.LineUp(2)
		if m.Chat.Selection.Active {
			m.Chat.Viewport.SetContent(m.applySelectionToLines(m.Chat.RenderedLines))
		}
		return m, nil
	}

	if msg.Button == tea.MouseButtonWheelDown {
		if m.Chat.Viewport.AtBottom() {
			m.Chat.AutoScroll = true
			return m, nil
		}
		m.Chat.Viewport.LineDown(2)
		if m.Chat.Viewport.AtBottom() {
			m.Chat.AutoScroll = true
		}
		if m.Chat.Selection.Active {
			m.Chat.Viewport.SetContent(m.applySelectionToLines(m.Chat.RenderedLines))
		}
		return m, nil
	}

	if len(m.Chat.RenderedLines) == 0 {
		_ = (&m).renderConversation()
	}

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}

		if msg.Y >= m.Chat.Viewport.Height {
			if !m.Chat.TextArea.Focused() {
				m.Chat.TextArea.Focus()
			}
			m.Chat.Selection.Active = false
			m.Chat.Selection.Dragging = false
			m.Chat.Viewport.SetContent(m.renderConversation())
			return m, textarea.Blink
		}

		m.Chat.AutoScroll = false
		absRow := m.Chat.Viewport.YOffset + msg.Y
		col := max(0, msg.X)
		m.Chat.Selection = TextSelection{
			Active:   true,
			Dragging: true,
			Start:    SelectionPoint{Row: absRow, Col: col},
			End:      SelectionPoint{Row: absRow, Col: col},
		}
		m.Chat.Viewport.SetContent(m.applySelectionToLines(m.Chat.RenderedLines))
		return m, nil

	case tea.MouseActionMotion:
		if !m.Chat.Selection.Dragging {
			return m, nil
		}

		if msg.Y <= 0 {
			m.Chat.Viewport.LineUp(1)
		} else if msg.Y >= m.Chat.Viewport.Height-1 {
			m.Chat.Viewport.LineDown(1)
		}

		clampedY := max(0, min(msg.Y, m.Chat.Viewport.Height-1))
		absRow := m.Chat.Viewport.YOffset + clampedY
		col := max(0, msg.X)

		m.Chat.Selection.End = SelectionPoint{Row: absRow, Col: col}
		m.Chat.Viewport.SetContent(m.applySelectionToLines(m.Chat.RenderedLines))
		return m, nil

	case tea.MouseActionRelease:
		if !m.Chat.Selection.Dragging {
			return m, nil
		}

		clampedY := max(0, min(msg.Y, m.Chat.Viewport.Height-1))
		absRow := m.Chat.Viewport.YOffset + clampedY
		col := max(0, msg.X)
		m.Chat.Selection.End = SelectionPoint{Row: absRow, Col: col}

		m.Chat.Selection.Dragging = false
		if m.Chat.Selection.IsEmpty() {
			m.Chat.Selection.Active = false
			m.Chat.Viewport.SetContent(m.renderConversation())
			return m, nil
		}

		newModel, copyCmd := m.copySelectedText()
		return newModel, copyCmd
	}

	return m, nil
}

func (m Model) applySelectionToLines(lines []string) string {
	if !m.Chat.Selection.Active || m.Chat.Selection.IsEmpty() || len(lines) == 0 {
		return strings.Join(lines, "\n")
	}

	start, end := m.Chat.Selection.Normalize()
	if start.Row >= len(lines) {
		return strings.Join(lines, "\n")
	}

	out := make([]string, len(lines))
	copy(out, lines)

	firstRow := max(0, start.Row)
	lastRow := min(len(lines)-1, end.Row)

	for r := firstRow; r <= lastRow; r++ {
		lines[r] = strings.TrimRight(lines[r], "\r")
		line := lines[r]
		lineWidth := ansi.StringWidth(line)
		if lineWidth == 0 {
			continue
		}

		var meta LineMeta
		if r < len(m.Chat.LineMetas) {
			meta = m.Chat.LineMetas[r]
		}
		if meta.IsDecoration {
			continue
		}

		sc := meta.ContentColStart
		ec := meta.ContentColEnd
		if ec <= 0 {
			plain := ansi.Strip(line)
			ec = ansi.StringWidth(strings.TrimRight(plain, " \r\n"))
		}

		if r == start.Row {
			sc = max(sc, start.Col)
		}
		if r == end.Row {
			ec = min(ec, end.Col)
		}

		if sc >= ec {
			continue
		}

		out[r] = highlightLineRange(line, sc, ec, lineWidth)
	}

	return strings.Join(out, "\n")
}

func highlightLineRange(line string, sc, ec, lineWidth int) string {
	line = strings.TrimRight(line, "\r")
	left := ansi.Cut(line, 0, sc)
	mid := ansi.Cut(line, sc, ec)
	right := ansi.Cut(line, ec, lineWidth)

	if mid == "" {
		return line
	}

	styledMid := strings.ReplaceAll(mid, "\x1b[0m", "\x1b[0m\x1b[7m")
	styledMid = strings.ReplaceAll(styledMid, "\x1b[m", "\x1b[m\x1b[7m")

	return left + "\x1b[7m" + styledMid + "\x1b[27m" + right
}

func (m Model) extractSelectedText() string {
	if !m.Chat.Selection.Active || m.Chat.Selection.IsEmpty() || len(m.Chat.RenderedLines) == 0 {
		return ""
	}

	start, end := m.Chat.Selection.Normalize()
	if start.Row >= len(m.Chat.RenderedLines) {
		return ""
	}

	firstRow := max(0, start.Row)
	lastRow := min(len(m.Chat.RenderedLines)-1, end.Row)

	var extracted []string

	for r := firstRow; r <= lastRow; r++ {
		line := m.Chat.RenderedLines[r]
		plain := ansi.Strip(line)
		trimmed := strings.TrimRight(plain, " \r\n")
		if len(trimmed) == 0 {
			if r > firstRow && r < lastRow {
				extracted = append(extracted, "")
			}
			continue
		}

		var meta LineMeta
		if r < len(m.Chat.LineMetas) {
			meta = m.Chat.LineMetas[r]
		}
		if meta.IsDecoration {
			continue
		}

		sc := meta.ContentColStart
		ec := meta.ContentColEnd
		if ec <= 0 {
			ec = ansi.StringWidth(trimmed)
		}
		if r == start.Row {
			sc = max(sc, start.Col)
		}
		if r == end.Row {
			ec = min(ec, end.Col)
		}
		if sc >= ec {
			continue
		}

		selectedPart := ansi.Cut(plain, sc, ec)
		selectedPart = strings.TrimRight(selectedPart, "\r\n")
		extracted = append(extracted, selectedPart)
	}

	return strings.Join(extracted, "\n")
}

func (m Model) copySelectedText() (Model, tea.Cmd) {
	text := m.extractSelectedText()
	if strings.TrimSpace(text) == "" {
		m.Chat.Selection.Active = false
		m.Chat.Viewport.SetContent(m.renderConversation())
		return m, nil
	}

	m.StatusBarMessage = "Selected text copied to clipboard"
	cfg := m.Session.GetConfig()
	var copyCmdStr string
	if cfg != nil {
		copyCmdStr = cfg.Clipboard.CopyCmd
	}

	return m, tea.Batch(
		copyToClipboardCmd(text, copyCmdStr),
		clearStatusBarCmd(),
	)
}

func copyToClipboardCmd(text, customCmd string) tea.Cmd {
	return func() tea.Msg {
		_ = clipboard.Copy(text, customCmd)
		emitOSC52(text)
		return nil
	}
}

func emitOSC52(text string) {
	b64 := base64.StdEncoding.EncodeToString([]byte(text))
	osc := fmt.Sprintf("\x1b]52;c;%s\x07", b64)
	if os.Getenv("TMUX") != "" {
		osc = fmt.Sprintf("\x1bPtmux;\x1b\x1b]52;c;%s\x07\x1b\\", b64)
	}
	_, _ = os.Stderr.WriteString(osc)
}
