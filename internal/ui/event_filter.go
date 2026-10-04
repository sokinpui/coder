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
	return func(_ tea.Model, msg tea.Msg) tea.Msg {
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
