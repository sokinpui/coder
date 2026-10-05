package ui

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var sgrFragmentRegex = regexp.MustCompile(`\[<\d+;\d+;\d+[Mm]`)

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

		raw := keyMsg.String()
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 {
			raw = string(keyMsg.Runes)
		}

		combined := filter.pending + raw
		filter.pending = ""

		cleaned := sgrFragmentRegex.ReplaceAllString(combined, "")

		if idx := strings.LastIndex(cleaned, "[<"); idx != -1 {
			filter.pending = cleaned[idx:]
			cleaned = cleaned[:idx]
		} else if strings.HasSuffix(cleaned, "[") {
			filter.pending = "["
			cleaned = cleaned[:len(cleaned)-1]
		}

		if len(filter.pending) > 32 {
			filter.pending = ""
		}

		if cleaned == "" {
			return nil
		}

		if cleaned == raw {
			return keyMsg
		}

		return tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune(cleaned),
		}
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
