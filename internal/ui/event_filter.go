package ui

import (
	"regexp"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	sgrFragmentRegex = regexp.MustCompile(`\[?<\d+;\d+;\d+[Mm]`)
	sgrPartialRegex  = regexp.MustCompile(`\[<\d*(?:;\d*)*$`)
	sgrMouseRegex    = regexp.MustCompile(`\[?<(\d+);(\d+);(\d+)([Mm])`)
)

func CleanSGRLeakedText(s string) string {
	if !sgrFragmentRegex.MatchString(s) {
		return s
	}
	return sgrFragmentRegex.ReplaceAllString(s, "")
}

type sgrSequenceFilter struct {
	pending string
}

func newSGRSequenceFilter() func(tea.Model, tea.Msg) tea.Msg {
	filter := &sgrSequenceFilter{}
	return func(tm tea.Model, msg tea.Msg) tea.Msg {
		if mouseMsg, ok := msg.(tea.MouseMsg); ok {
			if !canScrollMouse(tm, mouseMsg) {
				return nil
			}
			return mouseMsg
		}

		keyMsg, ok := msg.(tea.KeyMsg)
		if !ok {
			return msg
		}

		if keyMsg.Type != tea.KeyRunes || len(keyMsg.Runes) == 0 {
			filter.pending = ""
			return keyMsg
		}

		raw := string(keyMsg.Runes)
		if keyMsg.Alt && raw == "[" {
			filter.pending = "["
			return nil
		}

		combined := filter.pending + raw
		filter.pending = ""

		var mouseEv tea.Msg
		var isMouse bool
		if sgrFragmentRegex.MatchString(combined) {
			mouseEv, isMouse = parseSGRMouseMsg(combined)
			combined = sgrFragmentRegex.ReplaceAllString(combined, "")
		}

		cleaned := combined
		combined = ""

		if loc := sgrPartialRegex.FindStringIndex(cleaned); loc != nil {
			filter.pending = cleaned[loc[0]:]
			cleaned = cleaned[:loc[0]]
		}

		if len(filter.pending) > 32 {
			filter.pending = ""
		}

		if isMouse && mouseEv != nil {
			return mouseEv
		}

		if cleaned == "" {
			return nil
		}

		if cleaned == raw && !keyMsg.Alt {
			return keyMsg
		}

		return tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune(cleaned),
		}
	}
}

func parseSGRMouseMsg(s string) (tea.Msg, bool) {
	match := sgrMouseRegex.FindStringSubmatch(s)
	if len(match) < 5 {
		return nil, false
	}

	btn, _ := strconv.Atoi(match[1])
	x, _ := strconv.Atoi(match[2])
	y, _ := strconv.Atoi(match[3])

	if x > 0 {
		x--
	}
	if y > 0 {
		y--
	}

	switch btn {
	case 64:
		return tea.MouseMsg{
			X:      x,
			Y:      y,
			Button: tea.MouseButtonWheelUp,
			Action: tea.MouseActionPress,
		}, true
	case 65:
		return tea.MouseMsg{
			X:      x,
			Y:      y,
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		}, true
	default:
		return nil, true
	}
}

func canScrollMouse(tm tea.Model, msg tea.MouseMsg) bool {
	if msg.Button != tea.MouseButtonWheelUp && msg.Button != tea.MouseButtonWheelDown {
		return true
	}

	main := extractMainModel(tm)
	if main == nil {
		return true
	}

	switch main.ActiveOverlay {
	case overlayNone:
		if msg.Button == tea.MouseButtonWheelUp {
			return main.Chat.Viewport.YOffset > 0
		}
		return !main.Chat.Viewport.AtBottom()

	case overlayQuickView:
		if main.QuickView == nil {
			return false
		}
		if main.QuickView.needsRender {
			return true
		}
		if msg.Button == tea.MouseButtonWheelUp {
			return main.QuickView.Viewport.YOffset > 0
		}
		return !main.QuickView.Viewport.AtBottom()

	default:
		return false
	}
}

func extractMainModel(tm tea.Model) *Model {
	switch m := tm.(type) {
	case *Manager:
		if m != nil {
			return m.Main
		}
	case *Model:
		return m
	}
	return nil
}
