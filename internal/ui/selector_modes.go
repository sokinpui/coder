package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/clipboard"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/types"
)

type SelectorConfig struct {
	Title          string
	Placeholder    string
	Footer         string
	Items          []SelectorItem
	ShowSearch     bool
	IsSearching    bool
	MultiSelect    bool
	Tabs           []string
	ActiveTab      int
	InitialCursor  int
	OnConfirm      func(m Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd)
	OnCancel       func(m Model) (tea.Model, tea.Cmd)
	OnTabChange    func(m Model, newTab int) (tea.Model, tea.Cmd)
	OnCursorChange func(m Model, current *SelectorItem) Model
	KeyHandler     func(m Model, key tea.KeyMsg) (tea.Model, tea.Cmd, bool)
}

func (m Model) openGenericSelector(cfg SelectorConfig) (Model, tea.Cmd) {
	m.ActiveOverlay = overlaySelector
	m.Chat.TextArea.Blur()

	m.Selector = NewSelector()
	m.Selector.Title = cfg.Title
	m.Selector.MultiSelect = cfg.MultiSelect
	m.Selector.ShowSearch = cfg.ShowSearch
	m.Selector.IsSearching = cfg.IsSearching
	m.Selector.FooterHelp = cfg.Footer
	m.Selector.Tabs = cfg.Tabs
	m.Selector.ActiveTab = cfg.ActiveTab
	m.Selector.OnTabChange = cfg.OnTabChange
	m.Selector.OnCursorChange = cfg.OnCursorChange
	m.Selector.KeyHandler = cfg.KeyHandler
	m.Selector.OnConfirm = cfg.OnConfirm

	onCancel := cfg.OnCancel
	if onCancel == nil {
		onCancel = func(mod Model) (tea.Model, tea.Cmd) {
			mod.ActiveOverlay = overlayNone
			if mod.State == stateIdle {
				mod.Chat.TextArea.Focus()
				return mod, textarea.Blink
			}
			return mod, nil
		}
	}
	m.Selector.OnCancel = onCancel

	if cfg.Placeholder != "" {
		m.Selector.SearchInput.Placeholder = cfg.Placeholder
	}
	if cfg.ShowSearch {
		m.Selector.SearchInput.Focus()
	}

	m.Selector.SetItems(cfg.Items)

	if cfg.InitialCursor > 0 && cfg.InitialCursor < len(m.Selector.FilteredItems) {
		m.Selector.Cursor = cfg.InitialCursor
		m.Selector.Anchor = cfg.InitialCursor
	}

	if cfg.IsSearching {
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

	newModel, cmd := m.openGenericSelector(SelectorConfig{
		Title:       "── Switch Model ──",
		Placeholder: "Filter models...",
		Footer:      "── [Esc/Ctrl+C: cancel | Enter: select] ──",
		Items:       items,
		ShowSearch:  true,
		IsSearching: true,
		MultiSelect: false,
		OnConfirm: func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
			if primary == nil {
				mod.ActiveOverlay = overlayNone
				if mod.State == stateIdle {
					mod.Chat.TextArea.Focus()
				}
				return mod, textarea.Blink
			}

			modelName := primary.ID
			mod.Session.SetModel(modelName)
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
			return mod, textarea.Blink
		},
	})

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

	return m.openGenericSelector(SelectorConfig{
		Title:       title,
		Placeholder: placeholder,
		Footer:      "── [Esc/Ctrl+C: cancel | Tab: toggle select | Enter: apply] ──",
		Items:       items,
		ShowSearch:  true,
		IsSearching: true,
		MultiSelect: true,
		OnConfirm: func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
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
	})
}

func (m Model) openHistorySelector(initialTab int) (Model, tea.Cmd) {
	newModel, cmd := m.openGenericSelector(SelectorConfig{
		Tabs:        []string{"History", "Active"},
		ActiveTab:   initialTab,
		ShowSearch:  true,
		IsSearching: false,
		Footer:      "── [Esc/q: close | /: search | Tab/h/l: switch tab | Enter: load] ──",
		OnTabChange: func(mod Model, newTab int) (tea.Model, tea.Cmd) {
			mod.Selector.ActiveTab = newTab
			mod.Selector.Selected = make(map[string]struct{})
			mod.Selector.Cursor = 0
			mod.Selector.SearchInput.Reset()
			if newTab == 0 {
				return mod, listHistoryCmd(mod.Session.GetHistoryManager(), mod.Session.GetMode())
			}
			mod = mod.refreshHistorySelectorItems()
			return mod, nil
		},
		OnConfirm: func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
			if primary == nil {
				return mod, nil
			}

			mod.ActiveOverlay = overlayNone
			mod.Selector.IsSearching = false
			mod.Selector.SearchInput.Blur()

			if mod.Selector.ActiveTab == 1 {
				return mod, mod.switchSessionByID(primary.ID)
			}

			for _, sess := range mod.ActiveSessions {
				if sess.GetHistoryFilename() == primary.ID {
					return mod, mod.switchSessionByID(sess.GetID())
				}
			}

			return mod, loadConversationCmd(mod.Session, primary.ID)
		},
		OnCancel: func(mod Model) (tea.Model, tea.Cmd) {
			mod.ActiveOverlay = overlayNone
			mod.Selector.IsSearching = false
			mod.Selector.SearchInput.Blur()
			if mod.State == stateIdle {
				mod.Chat.TextArea.Focus()
				return mod, textarea.Blink
			}
			return mod, nil
		},
	})

	if initialTab == 1 {
		newModel = newModel.refreshHistorySelectorItems()
		return newModel, cmd
	}

	return newModel, listHistoryCmd(newModel.Session.GetHistoryManager(), newModel.Session.GetMode())
}

func (m Model) refreshHistorySelectorItems() Model {
	if m.Selector.ActiveTab != 1 {
		return m
	}
	var items []SelectorItem
	for i := len(m.ActiveSessions) - 1; i >= 0; i-- {
		sess := m.ActiveSessions[i]
		marker := ""
		if sess.GetID() == m.Session.GetID() {
			marker = "*"
		}
		items = append(items, SelectorItem{
			ID:          sess.GetID(),
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
		summary := getMessageSummary(msg)
		items = append(items, SelectorItem{
			ID:         fmt.Sprintf("%d", idx),
			Title:      summary,
			Badge:      badge,
			SearchText: msg.Content,
			Data:       idx,
		})
	}

	var actions []string
	if m.Session.Capabilities().Has(engine.CapITF) {
		actions = append(actions, "a")
	}
	actions = append(actions, "e")
	if m.Session.Capabilities().Has(engine.CapRegenerate) {
		actions = append(actions, "r")
	}
	if m.Session.Capabilities().Has(engine.CapBranch) {
		actions = append(actions, "b")
	}
	actionList := strings.Join(actions, "/")

	initialCursor := 0
	if len(items) > 0 {
		initialCursor = len(items) - 1
	}

	newModel, cmd := m.openGenericSelector(SelectorConfig{
		Title:         fmt.Sprintf("── Atomic Messages [Esc/C-c: exit | v: select | o: swap | y/d: copy/del | %s] ──", actionList),
		Items:         items,
		InitialCursor: initialCursor,
		OnCursorChange: func(mod Model, current *SelectorItem) Model {
			if current != nil {
				if idx, ok := current.Data.(int); ok {
					return mod.syncViewportToMessage(idx)
				}
			}
			return mod
		},
		OnCancel: func(mod Model) (tea.Model, tea.Cmd) {
			mod.ActiveOverlay = overlayNone
			mod.Selector.IsSelecting = false
			if mod.State == stateIdle {
				mod.Chat.TextArea.Focus()
				return mod, textarea.Blink
			}
			return mod, nil
		},
		KeyHandler: func(mod Model, keyMsg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
			return mod.handleAtomicMsgKey(keyMsg)
		},
	})

	newModel = newModel.updateLayout()
	return newModel, cmd
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
		if !m.Session.Capabilities().Has(engine.CapITF) {
			return m, nil, false
		}
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
		m.Session.AddMessages(types.Message{Type: types.FileApplyCmdMessage, Content: "/itf"})

		if res.Success {
			m.Session.AddMessages(types.Message{Type: types.FileApplyCmdResultMessage, Content: res.Summary})
		} else {
			m.Session.AddMessages(types.Message{Type: types.FileApplyCmdErrorMessage, Content: res.Summary})
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
			err := clipboard.CopyImage(imgMsg.Content, imgMsg.Data)
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
		_ = clipboard.Copy(combined, cfg.Clipboard.CopyCmd)
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
		if !targetMsg.Type.IsRegeneratable() || !m.Session.Capabilities().Has(engine.CapRegenerate) {
			m.StatusBarMessage = "Selected message cannot be regenerated."
			m.ActiveOverlay = overlayNone
			if m.State == stateIdle {
				m.Chat.TextArea.Focus()
			}
			return m, tea.Batch(clearStatusBarCmd(), textarea.Blink), true
		}

		if m.Chat.IsStreaming {
			m.Session.Cancel()
			m.Chat.IsStreaming = false
			m.Chat.EventSub = nil
		}

		m.ActiveOverlay = overlayNone
		m.Chat.TextArea.Focus()
		m.ClearCache()
		eventChan, err := m.Session.Regenerate(currIdx)
		if err != nil {
			m.StatusBarMessage = fmt.Sprintf("Error regenerating: %v", err)
			return m, clearStatusBarCmd(), true
		}
		model, cmd := m.startGenerationEvents(eventChan)
		return model, cmd, true

	case "d":
		var targetIndices []int
		for _, item := range m.Selector.GetSelectedItems() {
			targetIndices = append(targetIndices, item.Data.(int))
		}

		if m.Chat.IsStreaming {
			if slices.Contains(targetIndices, len(m.Session.GetMessages())-1) {
				m.Session.Cancel()
				m.Chat.IsStreaming = false
				m.Chat.EventSub = nil
			}
		}

		oldTotal := len(m.Session.GetMessages())
		m.Session.DeleteMessages(targetIndices)
		m.RemapCacheOnDelete(targetIndices, oldTotal)
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
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink, m.updateTokenCountCmd()), true

	case "b":
		if !m.Session.Capabilities().Has(engine.CapBranch) {
			return m, nil, false
		}
		m.Selector.IsSelecting = false
		if m.Chat.IsStreaming {
			m.Session.Cancel()
			m.Chat.IsStreaming = false
			m.Chat.EventSub = nil
		}

		oldSess := m.Session
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
		return m, tea.Batch(clearStatusBarCmd(), textarea.Blink, m.updateTokenCountCmd(), saveConversationCmd(oldSess)), true
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
		if msg.IsDocumentImage() {
			continue
		}
		if msg.Type.IsSelectable() {
			indices = append(indices, i)
		}
	}
	return indices
}

func getMessageSummary(msg types.Message) string {
	if msg.Type == types.ToolCallMessage && len(msg.ToolCalls) > 0 {
		var calls []string
		for _, tc := range msg.ToolCalls {
			name := tc.Name
			target := ExtractToolTarget(name, tc.Arguments)
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

func ExtractToolTarget(name, arguments string) string {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return ""
	}

	path := ExtractToolJSONField(trimmed, "path")
	if path != "" {
		return TruncateSingleLine(path, 40)
	}

	cmd := ExtractToolJSONField(trimmed, "command")
	if cmd != "" {
		return TruncateSingleLine(cmd, 40)
	}

	return SummarizeToolArgs(trimmed)
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
