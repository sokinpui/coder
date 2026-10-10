package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
	"sort"
)

const welcomeMessage = `Welcome to Coder!

- Press Enter for a new line in your prompt (or to run a command).
- Use Ctrl+J to send your message.
- Use Ctrl+E to edit your prompt in an external editor ($EDITOR).
- Use /model to open the model switcher.
- Use Ctrl+D and Ctrl+U to scroll the conversation.
- Use Ctrl+H to view conversation history.
- Press Ctrl+C twice to quit.
- During generation, press Ctrl+C to cancel.
- Type '/help' for a list of all commands and shortcuts.
`

type Model struct {
	Chat      ChatModel
	Selector  SelectorModel
	QuickView *QuickViewModel

	ActiveSessions      []engine.EngineSession
	Session             engine.EngineSession
	State               state
	ActiveOverlay       overlayMode
	Quitting            bool
	ConfirmRequest      *types.ToolConfirmRequest
	Height              int
	Width               int
	GlamourRenderer     *markdown.Renderer
	AvailableCommands   []string
	CommandDescriptions map[string]string
	StatusText          string
	StatusBarMessage    string
	TokenCount          int
	ToolsExpanded       bool
}

func NewModel(sess engine.EngineSession, initialInput string) (Model, error) {
	renderer, _ := markdown.NewRenderer(80)

	if len(sess.GetMessages()) == 0 {
		sess.AddMessages(types.Message{Type: types.InitMessage, Content: welcomeMessage})

		if dirMsg := project.DirInfo(); dirMsg != "" {
			sess.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})
		}
	}
	availableCommands := sess.GetSupportedCommands()
	commandDescriptions := sess.GetCommandDescriptions()
	sort.Strings(availableCommands)

	m := Model{
		ActiveSessions:      []engine.EngineSession{sess},
		Chat:                NewChat(initialInput),
		Selector:            NewSelector(),
		QuickView:           NewQuickView(),
		Session:             sess,
		ActiveOverlay:       overlayNone,
		State:               stateIdle,
		GlamourRenderer:     renderer,
		AvailableCommands:   availableCommands,
		CommandDescriptions: commandDescriptions,
	}
	return m, nil
}

func (m *Model) ClearCache() {
	m.Chat.RenderCache = make(map[int]markdown.CachedRender)
}

func (m *Model) PruneCacheAfter(idx int) {
	for k := range m.Chat.RenderCache {
		if k > idx {
			delete(m.Chat.RenderCache, k)
		}
	}
}

func (m *Model) RemapCacheOnDelete(deletedIndices []int, totalOld int) {
	toDelete := make(map[int]struct{}, len(deletedIndices))
	for _, idx := range deletedIndices {
		toDelete[idx] = struct{}{}
	}

	newRenderCache := make(map[int]markdown.CachedRender)
	newIdx := 0
	for oldIdx := range totalOld {
		if _, deleted := toDelete[oldIdx]; !deleted {
			if cached, ok := m.Chat.RenderCache[oldIdx]; ok {
				newRenderCache[newIdx] = cached
			}
			newIdx++
		}
	}
	m.Chat.RenderCache = newRenderCache
}

func (m Model) renderUncachedCmd() tea.Cmd {
	if m.Session == nil {
		return nil
	}
	return renderUncachedMessagesCmd(
		m.Session.GetID(),
		m.Session.GetMessages(),
		m.Chat.RenderCache,
		m.Chat.Viewport.Width,
		m.Chat.IsStreaming,
	)
}

func (m Model) updateTokenCountCmd() tea.Cmd {
	if m.Session == nil {
		return nil
	}
	sess := m.Session
	sessID := sess.GetID()
	return func() tea.Msg {
		count := sess.TokenCount()
		return tokenCountResultMsg{sessID: sessID, count: count}
	}
}

func (m *Model) addActiveSession(sess engine.EngineSession) {
	for i, s := range m.ActiveSessions {
		if s.GetID() == sess.GetID() {
			m.ActiveSessions[i] = sess
			return
		}
		// If a session with the same history file is already active, replace it.
		if sess.GetHistoryFilename() != "" && s.GetHistoryFilename() == sess.GetHistoryFilename() {
			m.ActiveSessions[i] = sess
			return
		}
	}
	m.ActiveSessions = append(m.ActiveSessions, sess)
}

func (m Model) switchSessionByID(id string) tea.Cmd {
	for _, sess := range m.ActiveSessions {
		if sess.GetID() == id {
			return func() tea.Msg { return switchActiveSessionMsg{sess: sess} }
		}
	}
	return nil
}

func (m Model) getSessionByID(id string) engine.EngineSession {
	if id == "" {
		return m.Session
	}
	if m.Session != nil && m.Session.GetID() == id {
		return m.Session
	}
	for _, s := range m.ActiveSessions {
		if s.GetID() == id {
			return s
		}
	}
	return nil
}

func (m Model) Keymap() config.Keymap {
	if m.Session != nil && m.Session.Capabilities().Has(engine.CapToolLoop) {
		return m.Session.GetConfig().Agent.Keymap
	}
	if m.Session != nil {
		return m.Session.GetConfig().Coder.Keymap
	}
	return config.DefaultKeymap()
}

func (m Model) needsSpinner() bool {
	if m.Chat.IsFetchingModels {
		return true
	}
	if m.ActiveOverlay == overlaySelector && m.Selector.IsLoading {
		return true
	}
	for _, s := range m.ActiveSessions {
		if s.IsStreaming() {
			return true
		}
	}
	switch m.State {
	case stateAsking, stateThinking, stateGenerating:
		return true
	default:
		return false
	}
}
