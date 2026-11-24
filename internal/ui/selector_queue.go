package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

const queueSelectorTag = "queue"

func (m Model) openQueueSelector() (Model, tea.Cmd) {
	prompts := m.currentQueuedPrompts()
	if len(prompts) == 0 {
		m.StatusBarMessage = "No queued messages."
		return m, clearStatusBarCmd()
	}

	items := buildQueueSelectorItems(prompts)

	return m.openGenericSelector(SelectorConfig{
		Tag:            queueSelectorTag,
		Title:          "Queued Messages [Esc/q: close | /: search | d: delete | Enter: send (idle) | C-d/C-u: scroll]",
		Placeholder:    "Filter queued messages...",
		Items:          items,
		ShowSearch:     true,
		IsSearching:    false,
		MultiSelect:    false,
		ShowPreview:    true,
		PreviewTitle:   "Queued Message #1",
		PreviewContent: prompts[0],
		CustomWidth:    max(60, min((m.Width*9)/10, m.Width-4)),
		CustomHeight:   max(18, min((m.Height*85)/100, m.Height-4)),
		OnCursorChange: func(mod Model, current *SelectorItem) Model {
			return mod.updateQueuePreview(current)
		},
		KeyHandler: func(mod Model, keyMsg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
			return mod.handleQueueKey(keyMsg)
		},
		OnConfirm: func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
			if primary == nil {
				mod.ActiveOverlay = overlayNone
				if mod.shouldFocusTextArea() {
					mod.Chat.TextArea.Focus()
					return mod, textarea.Blink
				}
				return mod, nil
			}
			if mod.State != stateIdle {
				return mod, nil
			}
			idx, ok := primary.Data.(int)
			current := mod.currentQueuedPrompts()
			if !ok || idx < 0 || idx >= len(current) {
				return mod, nil
			}
			target := current[idx]
			updated := append(append([]string(nil), current[:idx]...), current[idx+1:]...)
			mod.setQueuedPrompts(updated)
			mod.ActiveOverlay = overlayNone
			return mod.submitInput(target, false)
		},
	})
}

func (m Model) handleQueueKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.Selector.IsSearching {
		return m, nil, false
	}

	if msg.String() != "d" {
		return m, nil, false
	}

	primary := m.Selector.GetPrimaryItem()
	if primary == nil {
		return m, nil, true
	}

	idx, ok := primary.Data.(int)
	prompts := m.currentQueuedPrompts()
	if !ok || idx < 0 || idx >= len(prompts) {
		return m, nil, true
	}

	updated := append(append([]string(nil), prompts[:idx]...), prompts[idx+1:]...)
	m.setQueuedPrompts(updated)

	if len(updated) == 0 {
		m.ActiveOverlay = overlayNone
		m.StatusBarMessage = "Deleted queued message."
		if m.shouldFocusTextArea() {
			m.Chat.TextArea.Focus()
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}
		return m, clearStatusBarCmd(), true
	}

	m = m.refreshQueueSelector()
	return m, nil, true
}

func (m Model) refreshQueueSelector() Model {
	if m.ActiveOverlay != overlaySelector || m.Selector.Tag != queueSelectorTag {
		return m
	}
	prompts := m.currentQueuedPrompts()
	if len(prompts) == 0 {
		m.Selector.SetItems(nil)
		m.Selector.PreviewTitle = "Queue"
		m.Selector.PreviewDescription = ""
		m.Selector.PreviewContent = "No queued messages."
		m.Selector.PreviewOffset = 0
		return m
	}

	m.Selector.SetItems(buildQueueSelectorItems(prompts))
	return m.updateQueuePreview(m.Selector.GetPrimaryItem())
}

func (m Model) updateQueuePreview(current *SelectorItem) Model {
	if current == nil {
		m.Selector.PreviewTitle = ""
		m.Selector.PreviewContent = ""
		m.Selector.PreviewOffset = 0
		return m
	}
	idx, ok := current.Data.(int)
	prompts := m.currentQueuedPrompts()
	if !ok || idx < 0 || idx >= len(prompts) {
		return m
	}
	m.Selector.PreviewTitle = fmt.Sprintf("Queued Message #%d", idx+1)
	m.Selector.PreviewContent = prompts[idx]
	m.Selector.PreviewOffset = 0
	return m
}

func buildQueueSelectorItems(prompts []string) []SelectorItem {
	items := make([]SelectorItem, 0, len(prompts))
	for i, p := range prompts {
		items = append(items, SelectorItem{
			ID:         fmt.Sprintf("%d", i),
			Title:      getOneLineSummary(p),
			Badge:      fmt.Sprintf("[%02d]", i+1),
			SearchText: p,
			Data:       i,
		})
	}
	return items
}
