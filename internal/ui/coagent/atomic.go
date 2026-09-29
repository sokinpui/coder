package coagentui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/clipboard"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/types"
	coderui "github.com/sokinpui/coder/internal/ui/coder"
)

func (m Model) openAtomicMsgMode() (tea.Model, tea.Cmd) {
	if m.session == nil || len(m.session.Messages) == 0 {
		return m, nil
	}

	var items []coderui.SelectorItem
	for i, msg := range m.session.Messages {
		if !msg.Type.IsSelectable() {
			continue
		}
		badge := fmt.Sprintf("[%02d %s]", i+1, msg.Type.String())
		summary := getMessageSummary(msg)
		items = append(items, coderui.SelectorItem{
			ID:         fmt.Sprintf("%d", i),
			Title:      summary,
			Badge:      badge,
			SearchText: msg.Content,
			Data:       i,
		})
	}

	if len(items) == 0 {
		return m, nil
	}

	m.showSelector = true
	m.activeOverlay = overlaySelector
	m.input.Model.Blur()

	m.selector = coderui.NewSelector()
	m.selector.Title = "── Atomic Messages [Esc/C-c: exit | v: select | o: swap | y/d: copy/del | e: edit] ──"
	m.selector.ShowSearch = false
	m.selector.IsSearching = false
	m.selector.FooterHelp = ""
	m.selector.SetItems(items)
	m.selector.Cursor = len(items) - 1
	m.selector.Anchor = m.selector.Cursor

	m.selector.OnCursorChange = func(mod coderui.Model, current *coderui.SelectorItem) coderui.Model {
		return mod
	}

	if p := m.selector.GetPrimaryItem(); p != nil {
		if idx, ok := p.Data.(int); ok {
			m.viewport.SyncToMessage(idx)
		}
	}

	return m, nil
}

func (m Model) handleAtomicMsgKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	primary := m.selector.GetPrimaryItem()
	if primary == nil {
		m.showSelector = false
		m.activeOverlay = overlayNone
		m.input.Model.Focus()
		return m, nil
	}

	currIdx := primary.Data.(int)
	messages := m.session.Messages

	switch msg.String() {
	case "v":
		m.selector.IsSelecting = !m.selector.IsSelecting
		if m.selector.IsSelecting {
			m.selector.Anchor = m.selector.Cursor
		}
		return m, nil

	case "o", "O":
		if m.selector.IsSelecting {
			m.selector.Cursor, m.selector.Anchor = m.selector.Anchor, m.selector.Cursor
			if p := m.selector.GetPrimaryItem(); p != nil {
				m.viewport.SyncToMessage(p.Data.(int))
			}
		}
		return m, nil

	case "y":
		var targetIndices []int
		for _, item := range m.selector.GetSelectedItems() {
			targetIndices = append(targetIndices, item.Data.(int))
		}
		if len(targetIndices) == 1 && messages[targetIndices[0]].Type == types.ImageMessage {
			imgMsg := messages[targetIndices[0]]
			err := clipboard.CopyImage(imgMsg.Content, imgMsg.Data)
			if err != nil {
				m.statusBarMessage = fmt.Sprintf("Failed to copy image: %v", err)
			} else {
				m.statusBarMessage = "Image copied to clipboard."
			}
			m.showSelector = false
			m.activeOverlay = overlayNone
			m.input.Model.Focus()
			return m, clearStatusBarCmd()
		}
		var contents []string
		for _, idx := range targetIndices {
			contents = append(contents, messages[idx].Content)
		}
		_ = clipboard.Copy(strings.Join(contents, "\n\n"), m.cfg.Clipboard.CopyCmd)
		if len(targetIndices) > 1 {
			m.statusBarMessage = fmt.Sprintf("%d messages copied to clipboard.", len(targetIndices))
		} else {
			m.statusBarMessage = "Message copied to clipboard."
		}
		m.showSelector = false
		m.activeOverlay = overlayNone
		m.input.Model.Focus()
		return m, clearStatusBarCmd()

	case "d":
		var targetIndices []int
		for _, item := range m.selector.GetSelectedItems() {
			targetIndices = append(targetIndices, item.Data.(int))
		}
		toDelete := make(map[int]struct{}, len(targetIndices))
		for _, idx := range targetIndices {
			toDelete[idx] = struct{}{}
		}
		var remaining []types.Message
		for i, msgItem := range m.session.Messages {
			if _, del := toDelete[i]; !del {
				remaining = append(remaining, msgItem)
			}
		}
		m.session.Messages = remaining
		_ = m.session.SaveConversation()
		m.tokenCount = token.CountTokens(m.session.GetPrompt())
		if len(targetIndices) > 1 {
			m.statusBarMessage = fmt.Sprintf("Deleted %d messages.", len(targetIndices))
		} else {
			m.statusBarMessage = "Deleted message."
		}
		m.showSelector = false
		m.activeOverlay = overlayNone
		m.input.Model.Focus()
		m.viewport.ClearCache()
		return m.updateViewportContent(), clearStatusBarCmd()

	case "e":
		m.selector.IsSelecting = false
		targetMsg := messages[currIdx]
		if !targetMsg.Type.IsEditable() {
			return m, nil
		}
		m.showSelector = false
		m.activeOverlay = overlayNone
		m.editingMsgIdx = currIdx
		return m, editInEditorCmd(targetMsg.Content)

	case "gg":
		if len(m.selector.FilteredItems) > 0 {
			m.selector.Cursor = 0
			if p := m.selector.GetPrimaryItem(); p != nil {
				m.viewport.SyncToMessage(p.Data.(int))
			}
		}
		return m, nil

	case "G":
		if len(m.selector.FilteredItems) > 0 {
			m.selector.Cursor = len(m.selector.FilteredItems) - 1
			if p := m.selector.GetPrimaryItem(); p != nil {
				m.viewport.SyncToMessage(p.Data.(int))
			}
		}
		return m, nil

	case "ctrl+d":
		m.viewport.HalfPageDown()
		return m, nil

	case "ctrl+u":
		m.viewport.HalfPageUp()
		return m, nil

	case "j":
		if len(m.selector.FilteredItems) > 0 {
			m.selector.Cursor = min(len(m.selector.FilteredItems)-1, m.selector.Cursor+1)
			if p := m.selector.GetPrimaryItem(); p != nil {
				m.viewport.SyncToMessage(p.Data.(int))
			}
		}
		return m, nil

	case "k":
		if len(m.selector.FilteredItems) > 0 {
			m.selector.Cursor = max(0, m.selector.Cursor-1)
			if p := m.selector.GetPrimaryItem(); p != nil {
				m.viewport.SyncToMessage(p.Data.(int))
			}
		}
		return m, nil

	case "u":
		m.viewport.HalfPageUp()
		return m, nil

	case "q":
		m.showSelector = false
		m.activeOverlay = overlayNone
		m.input.Model.Focus()
		return m, nil
	}

	switch msg.Type {
	case tea.KeyUp, tea.KeyCtrlK, tea.KeyCtrlP:
		if len(m.selector.FilteredItems) > 0 {
			m.selector.Cursor = max(0, m.selector.Cursor-1)
			if p := m.selector.GetPrimaryItem(); p != nil {
				m.viewport.SyncToMessage(p.Data.(int))
			}
		}
		return m, nil
	case tea.KeyDown, tea.KeyCtrlJ, tea.KeyCtrlN:
		if len(m.selector.FilteredItems) > 0 {
			m.selector.Cursor = min(len(m.selector.FilteredItems)-1, m.selector.Cursor+1)
			if p := m.selector.GetPrimaryItem(); p != nil {
				m.viewport.SyncToMessage(p.Data.(int))
			}
		}
		return m, nil
	}

	return m, nil
}

func getMessageSummary(msg types.Message) string {
	if msg.Type == types.ToolCallMessage && len(msg.ToolCalls) > 0 {
		var calls []string
		for _, tc := range msg.ToolCalls {
			name := tc.Name
			target := coderui.ExtractToolTarget(name, tc.Arguments)
			if target != "" {
				calls = append(calls, fmt.Sprintf("%s %s", name, target))
			} else {
				calls = append(calls, name)
			}
		}
		return strings.Join(calls, ", ")
	}

	if msg.Type == types.ToolResultMessage {
		trimmed := strings.TrimSpace(msg.Content)
		if !strings.HasPrefix(trimmed, "Error:") && !strings.HasPrefix(trimmed, "cannot read") {
			lines := strings.Split(strings.TrimRight(msg.Content, "\r\n"), "\n")
			if len(lines) > 1 {
				return fmt.Sprintf("%d lines read", len(lines))
			}
		}
	}

	return getOneLineSummary(msg.Content)
}

func getOneLineSummary(content string) string {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return "(empty)"
}
