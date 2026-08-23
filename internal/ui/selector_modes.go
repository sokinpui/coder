package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
)

func (m Model) openGenericSelector(items []SelectorItem, title, placeholder, footer string, showSearch bool, onConfirm func(m Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd)) (Model, tea.Cmd) {
	m.ActiveOverlay = overlaySelector
	m.Chat.TextArea.Blur()

	m.Selector = NewSelector()
	m.Selector.Title = title
	m.Selector.ShowSearch = showSearch
	m.Selector.IsSearching = showSearch
	m.Selector.FooterHelp = footer
	m.Selector.OnConfirm = onConfirm
	m.Selector.OnCancel = func(mod Model) (tea.Model, tea.Cmd) {
		mod.ActiveOverlay = overlayNone
		if mod.State == stateIdle {
			mod.Chat.TextArea.Focus()
			return mod, textarea.Blink
		}
		return mod, nil
	}

	if placeholder != "" {
		m.Selector.SearchInput.Placeholder = placeholder
	}
	if showSearch {
		m.Selector.SearchInput.Focus()
	}

	m.Selector.SetItems(items)
	m.UpdateTokenCount()

	if showSearch {
		return m, textinput.Blink
	}
	return m, nil
}

func (m Model) openModelSelector(initialQuery string) (Model, tea.Cmd) {
	cfg := m.Session.GetConfig()
	var items []SelectorItem
	for _, modelName := range cfg.AvailableModels {
		items = append(items, SelectorItem{
			ID:    modelName,
			Title: modelName,
		})
	}

	newModel, cmd := m.openGenericSelector(
		items,
		"── Switch Model ──",
		"Filter models...",
		"── [Esc/Ctrl+C: cancel | Enter: select] ──",
		true,
		func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
			if primary == nil {
				mod.ActiveOverlay = overlayNone
				if mod.State == stateIdle {
					mod.Chat.TextArea.Focus()
				}
				return mod, textarea.Blink
			}

			modelName := primary.ID
			c := mod.Session.GetConfig()
			c.Generation.ModelCode = modelName
			mod.Session.AddMessages(
				types.Message{Type: types.CommandMessage, Content: "/model " + modelName},
				types.Message{Type: types.CommandResultMessage, Content: fmt.Sprintf("Switched model to: %s", modelName)},
			)
			mod.ActiveOverlay = overlayNone
			mod.Chat.Viewport.SetContent(mod.renderConversation())
			mod.Chat.Viewport.GotoBottom()
			if mod.State == stateIdle {
				mod.Chat.TextArea.Focus()
			}
			mod.UpdateTokenCount()
			return mod, textarea.Blink
		},
	)

	if initialQuery != "" {
		newModel.Selector.SearchInput.SetValue(initialQuery)
		newModel.Selector.UpdateFilter()
	}
	return newModel, cmd
}

func (m Model) openFileListSelector(title, placeholder string, paths []string, onApply func(mod Model, selectedPaths []string) (tea.Model, tea.Cmd)) (Model, tea.Cmd) {
	var items []SelectorItem
	for _, p := range paths {
		items = append(items, SelectorItem{
			ID:    p,
			Title: p,
		})
	}

	return m.openGenericSelector(
		items,
		title,
		placeholder,
		"── [Esc/Ctrl+C: cancel | Tab: toggle select | Enter: apply] ──",
		true,
		func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
			var picked []string
			for _, item := range selected {
				picked = append(picked, item.ID)
			}
			mod.ActiveOverlay = overlayNone
			if len(picked) == 0 {
				if mod.State == stateIdle {
					mod.Chat.TextArea.Focus()
				}
				return mod, textarea.Blink
			}
			return onApply(mod, picked)
		},
	)
}

func (m Model) openHistorySelector(initialTab int) (Model, tea.Cmd) {
	m.ActiveOverlay = overlaySelector
	m.Chat.TextArea.Blur()

	m.Selector = NewSelector()
	m.Selector.Tabs = []string{"History", "Active"}
	m.Selector.ActiveTab = initialTab
	m.Selector.ShowSearch = true
	m.Selector.IsSearching = false
	m.Selector.FooterHelp = "── [Esc/q: close | /: search | Tab/h/l: switch tab | Enter: load] ──"

	m.Selector.OnTabChange = func(mod Model, newTab int) (tea.Model, tea.Cmd) {
		mod.Selector.ActiveTab = newTab
		mod.Selector.Selected = make(map[string]struct{})
		mod.Selector.Cursor = 0
		mod.Selector.SearchInput.Reset()
		if newTab == 0 {
			return mod, listHistoryCmd(mod.Session.GetHistoryManager())
		}
		mod = mod.refreshHistorySelectorItems()
		return mod, nil
	}

	m.Selector.OnConfirm = func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
		if primary == nil {
			return mod, nil
		}

		if mod.Chat.IsStreaming {
			mod.Session.CancelGeneration()
			mod.Chat.IsStreaming = false
			mod.Chat.StreamSub = nil
		}

		mod.ActiveOverlay = overlayNone
		mod.Selector.IsSearching = false
		mod.Selector.SearchInput.Blur()

		if mod.Selector.ActiveTab == 1 { // Active tab
			return mod, mod.switchSessionByID(primary.ID)
		}

		for _, sess := range mod.ActiveSessions {
			if sess.GetHistoryFilename() == primary.ID {
				return mod, mod.switchSessionByID(sess.ID)
			}
		}

		return mod, loadConversationCmd(mod.Session, primary.ID)
	}

	m.Selector.OnCancel = func(mod Model) (tea.Model, tea.Cmd) {
		mod.ActiveOverlay = overlayNone
		mod.Selector.IsSearching = false
		mod.Selector.SearchInput.Blur()
		if mod.State == stateIdle {
			mod.Chat.TextArea.Focus()
			return mod, textarea.Blink
		}
		return mod, nil
	}

	if initialTab == 1 {
		m = m.refreshHistorySelectorItems()
		return m, nil
	}

	return m, listHistoryCmd(m.Session.GetHistoryManager())
}

func (m Model) refreshHistorySelectorItems() Model {
	if m.Selector.ActiveTab != 1 {
		return m
	}
	var items []SelectorItem
	for i := len(m.ActiveSessions) - 1; i >= 0; i-- {
		sess := m.ActiveSessions[i]
		marker := ""
		if sess.ID == m.Session.ID {
			marker = "*"
		}
		items = append(items, SelectorItem{
			ID:          sess.ID,
			Title:       sess.GetTitle(),
			Description: marker,
			Data:        sess,
		})
	}
	m.Selector.SetItems(items)
	return m
}

func (m Model) openAtomicMsgMode() (Model, tea.Cmd) {
	messages := m.Session.GetMessages()
	selectable := getSelectableIndices(messages)

	var items []SelectorItem
	for _, idx := range selectable {
		msg := messages[idx]
		badge := fmt.Sprintf("[%02d %s]", idx+1, msg.Type.String())
		summary := getOneLineSummary(msg.Content)
		items = append(items, SelectorItem{
			ID:         fmt.Sprintf("%d", idx),
			Title:      summary,
			Badge:      badge,
			SearchText: msg.Content,
			Data:       idx,
		})
	}

	m.ActiveOverlay = overlaySelector
	m.Chat.TextArea.Blur()

	m.Selector = NewSelector()
	m.Selector.Title = "── Atomic Messages [Esc/C-c: exit | v: select | o: swap | y/d: copy/del | a/e/r/b] ──"
	m.Selector.ShowSearch = false
	m.Selector.IsSearching = false
	m.Selector.FooterHelp = ""
	m.Selector.SetItems(items)

	if len(items) > 0 {
		m.Selector.Cursor = len(items) - 1
		m.Selector.Anchor = m.Selector.Cursor
		if primary := m.Selector.GetPrimaryItem(); primary != nil {
			m = m.syncViewportToMessage(primary.Data.(int))
		}
	}

	m.Selector.OnCursorChange = func(mod Model, current *SelectorItem) Model {
		if current != nil {
			if idx, ok := current.Data.(int); ok {
				return mod.syncViewportToMessage(idx)
			}
		}
		return mod
	}

	m.Selector.OnCancel = func(mod Model) (tea.Model, tea.Cmd) {
		mod.ActiveOverlay = overlayNone
		mod.Selector.IsSelecting = false
		if mod.State == stateIdle {
			mod.Chat.TextArea.Focus()
			return mod, textarea.Blink
		}
		return mod, nil
	}

	m.Selector.KeyHandler = func(mod Model, keyMsg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
		return mod.handleAtomicMsgKey(keyMsg)
	}

	m = m.updateLayout()
	return m, nil
}

func (m Model) handleAtomicMsgKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	primary := m.Selector.GetPrimaryItem()
	if primary == nil {
		return m, nil, false
	}
	currIdx := primary.Data.(int)
	messages := m.Session.GetMessages()

	switch msg.String() {
	case "v":
		if m.Selector.IsSelecting {
			m.Selector.IsSelecting = false
		} else {
			m.Selector.IsSelecting = true
			m.Selector.Anchor = m.Selector.Cursor
		}
		return m, nil, true

	case "o", "O":
		if m.Selector.IsSelecting {
			m.Selector.Cursor, m.Selector.Anchor = m.Selector.Anchor, m.Selector.Cursor
			if p := m.Selector.GetPrimaryItem(); p != nil {
				m = m.syncViewportToMessage(p.Data.(int))
			}
		}
		return m, nil, true

	case "ctrl+d":
		m.Chat.Viewport.HalfPageDown()
		return m, nil, true

	case "ctrl+u":
		m.Chat.Viewport.HalfPageUp()
		return m, nil, true

	case "a":
		m.Selector.IsSelecting = false
		targetMsg := messages[currIdx]
		var aiResponseToApply string

		if targetMsg.Type == types.AIMessage && targetMsg.Content != "" {
			aiResponseToApply = targetMsg.Content
		} else {
			for i := currIdx; i >= 0; i-- {
				if messages[i].Type == types.AIMessage && messages[i].Content != "" {
					aiResponseToApply = messages[i].Content
					break
				}
			}
		}

		if aiResponseToApply == "" {
			m.StatusBarMessage = "No AI response found to apply."
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		res := commands.ExecuteItf(aiResponseToApply, "")
		m.Session.SetLastModifiedFiles(res.AffectedFiles)
		m.Session.AddMessages(types.Message{Type: types.CommandMessage, Content: "/itf"})

		if res.Success {
			m.Session.AddMessages(types.Message{Type: types.CommandResultMessage, Content: res.Summary})
		} else {
			m.Session.AddMessages(types.Message{Type: types.CommandErrorResultMessage, Content: res.Summary})
		}

		m.ActiveOverlay = overlayNone
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		return m, textarea.Blink, true

	case "y":
		var targetIndices []int
		for _, item := range m.Selector.GetSelectedItems() {
			targetIndices = append(targetIndices, item.Data.(int))
		}

		if len(targetIndices) == 1 && messages[targetIndices[0]].Type == types.ImageMessage {
			imgMsg := messages[targetIndices[0]]
			err := utils.CopyImage(imgMsg.Content, imgMsg.Data)
			if err != nil {
				m.StatusBarMessage = fmt.Sprintf("Failed to copy image: %v", err)
			} else {
				m.StatusBarMessage = "Image copied to clipboard."
			}
			m.ActiveOverlay = overlayNone
			m.Selector.IsSelecting = false
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		var contents []string
		for _, idx := range targetIndices {
			contents = append(contents, messages[idx].Content)
		}
		combined := strings.Join(contents, "\n\n")
		cfg := m.Session.GetConfig()
		_ = utils.Copy(combined, cfg.Clipboard.CopyCmd)
		if len(targetIndices) > 1 {
			m.StatusBarMessage = fmt.Sprintf("%d messages copied to clipboard.", len(targetIndices))
		} else {
			m.StatusBarMessage = "Message copied to clipboard."
		}
		m.ActiveOverlay = overlayNone
		m.Selector.IsSelecting = false
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true

	case "e":
		m.Selector.IsSelecting = false
		targetMsg := messages[currIdx]
		if !targetMsg.Type.IsEditable() {
			m.StatusBarMessage = "Only user messages can be edited."
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		m.ActiveOverlay = overlayNone
		m.Chat.EditingMessageIndex = currIdx
		return m, editInEditorCmd(targetMsg.Content), true

	case "r":
		m.Selector.IsSelecting = false
		targetMsg := messages[currIdx]
		if !targetMsg.Type.IsRegeneratable() {
			m.StatusBarMessage = "Selected message cannot be regenerated."
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false
			m.Chat.StreamSub = nil
		}

		m.ActiveOverlay = overlayNone
		m.Chat.TextArea.Focus()
		event := m.Session.RegenerateFrom(currIdx)
		model, cmd := m.startGeneration(event)
		return model, cmd, true

	case "d":
		var targetIndices []int
		for _, item := range m.Selector.GetSelectedItems() {
			targetIndices = append(targetIndices, item.Data.(int))
		}

		if m.Chat.IsStreaming {
			if slices.Contains(targetIndices, len(m.Session.GetMessages())-1) {
				m.Session.CancelGeneration()
				m.Chat.IsStreaming = false
				m.Chat.StreamSub = nil
			}
		}

		m.Session.DeleteMessages(targetIndices)
		m.ClearCache()
		m.UpdateTokenCount()
		if len(targetIndices) > 1 {
			m.StatusBarMessage = fmt.Sprintf("Deleted %d messages.", len(targetIndices))
		} else {
			m.StatusBarMessage = "Deleted message."
		}
		m.ActiveOverlay = overlayNone
		m.Selector.IsSelecting = false
		if m.State == stateIdle {
			m.Chat.TextArea.Focus()
		}
		m.Chat.Viewport.SetContent(m.renderConversation())
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true

	case "b":
		m.Selector.IsSelecting = false
		if m.Chat.IsStreaming {
			m.Session.CancelGeneration()
			m.Chat.IsStreaming = false
			m.Chat.StreamSub = nil
		}

		newSess, err := m.Session.Branch(currIdx)
		if err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error branching: %v", err)
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		m.ActiveOverlay = overlayNone
		m.Session = newSess
		m.addActiveSession(newSess)
		m.StatusBarMessage = "Branched to a new session."
		m.Chat.LastInteractionFailed = false
		m.Chat.TextArea.Reset()
		m.Chat.TextArea.SetHeight(1)
		m.Chat.TextArea.Focus()
		m.Chat.Viewport.SetContent(m.renderConversation())
		m.Chat.Viewport.GotoBottom()
		m.UpdateTokenCount()
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
	}

	return m, nil, false
}

func (m Model) syncViewportToMessage(msgIdx int) Model {
	if line, ok := m.Chat.MessageLineOffsets[msgIdx]; ok {
		viewportHeight := m.Chat.Viewport.Height
		targetY := max(line-(viewportHeight/3), 0)
		m.Chat.Viewport.SetYOffset(targetY)
	}
	return m
}

func getSelectableIndices(messages []types.Message) []int {
	var indices []int
	for i, msg := range messages {
		if msg.Type.IsSelectable() {
			indices = append(indices, i)
		}
	}
	return indices
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
