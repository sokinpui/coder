package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/engine"
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

func (m Model) openReasoningSelector() (Model, tea.Cmd) {
	levels := []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}
	var items []SelectorItem
	currentEffort := m.Session.GetReasoningEffort()
	initialCursor := 0
	for i, lvl := range levels {
		badge := " "
		if strings.EqualFold(lvl, currentEffort) {
			badge = "*"
			initialCursor = i
		}
		items = append(items, SelectorItem{
			ID:    lvl,
			Title: lvl,
			Badge: badge,
		})
	}

	return m.openGenericSelector(SelectorConfig{
		Title:         "── Switch Reasoning Effort ──",
		Footer:        "── [Esc/Ctrl+C: cancel | Enter: select] ──",
		Items:         items,
		ShowSearch:    false,
		IsSearching:   false,
		MultiSelect:   false,
		InitialCursor: initialCursor,
		OnConfirm: func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
			if primary == nil {
				mod.ActiveOverlay = overlayNone
				if mod.State == stateIdle {
					mod.Chat.TextArea.Focus()
				}
				return mod, textarea.Blink
			}

			effort := primary.ID
			mod.Session.SetReasoningEffort(effort)
			mod.Session.AddMessages(
				types.Message{Type: types.CommandMessage, Content: "/reasoning " + effort},
				types.Message{Type: types.CommandResultMessage, Content: fmt.Sprintf("Switched reasoning effort to: %s", effort)},
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
}

func (m Model) openModeSelector() (Model, tea.Cmd) {
	currentMode := m.Session.GetMode()
	modes := []struct {
		id   string
		name string
		desc string
	}{
		{id: engine.ModeAgent, name: "[agent]", desc: "Autonomous agent with tools"},
		{id: engine.ModeCoder, name: "[coder]", desc: "Code editor with context"},
		{id: engine.ModeChat, name: "[chat] ", desc: "General chat without context"},
	}

	var items []SelectorItem
	initialCursor := 0
	for i, md := range modes {
		badge := " "
		if md.id == currentMode {
			badge = "*"
			initialCursor = i
		}
		items = append(items, SelectorItem{
			ID:          md.id,
			Title:       md.name,
			Description: "  " + md.desc,
			Badge:       badge,
		})
	}

	return m.openGenericSelector(SelectorConfig{
		Title:         "── Switch Mode ──",
		Footer:        "── [Esc/Ctrl+C: cancel | Enter: select] ──",
		Items:         items,
		ShowSearch:    false,
		IsSearching:   false,
		MultiSelect:   false,
		InitialCursor: initialCursor,
		OnConfirm: func(mod Model, selected []SelectorItem, primary *SelectorItem) (tea.Model, tea.Cmd) {
			if primary == nil || primary.ID == mod.Session.GetMode() {
				mod.ActiveOverlay = overlayNone
				if mod.State == stateIdle {
					mod.Chat.TextArea.Focus()
				}
				return mod, textarea.Blink
			}

			mod.ActiveOverlay = overlayNone
			newModel, cmd := mod.newSession(primary.ID)
			return newModel, cmd
		},
	})
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
