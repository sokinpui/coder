package coderui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/coder"
	"github.com/sokinpui/coder/internal/engine/commands"
	"github.com/sokinpui/coder/internal/engine/token"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"sort"

	"github.com/charmbracelet/glamour"
)

const welcomeMessage = `Welcome to Coder!

- Press Enter for a new line in your prompt (or to run a command).
- Use Ctrl+J to send your message.
- Use Ctrl+E to edit your prompt in an external editor ($EDITOR).
- Use /model to open the model switcher.
- Use Ctrl+D and Ctrl+U to scroll the conversation.
- Use Ctrl+H to view conversation history.
- Use Esc or Ctrl+C to clear the input. Press Ctrl+C again on an empty line to quit.
- During generation, press Ctrl+C to cancel.
- Type '/help' for a list of all commands and shortcuts.
`

type Model struct {
	Chat      ChatModel
	Selector  SelectorModel
	QuickView *QuickViewModel

	ActiveSessions      []*coder.Session
	Session             *coder.Session
	State               state
	ActiveOverlay       overlayMode
	Quitting            bool
	Height              int
	Width               int
	GlamourRenderer     *glamour.TermRenderer
	AvailableCommands   []string
	CommandDescriptions map[string]string
	StatusBarMessage    string
	TokenCount          int
}

func NewModel(cfg *config.Config, mode string, initialInput string, contextFiles []string, instruction string) (Model, error) {
	sess, err := coder.New(cfg, mode, instruction, contextFiles)
	if err != nil {
		return Model{}, err
	}
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle(cfg.UI.MarkdownTheme),
		glamour.WithWordWrap(80),
	)

	sess.AddMessages(types.Message{Type: types.InitMessage, Content: welcomeMessage})

	dirMsg := project.DirInfo()
	sess.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})
	availableCommands := commands.GetCommands()
	commandDescriptions := commands.GetCommandDescriptions()
	sort.Strings(availableCommands)

	m := Model{
		ActiveSessions:      []*coder.Session{sess},
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
	m.Chat.RenderCache = make(map[int]cachedRender)
}

func (m Model) updateTokenCountCmd() tea.Cmd {
	if m.Session == nil {
		return nil
	}
	sessID := m.Session.ID
	promptMsgs := m.Session.GetPrompt()
	return func() tea.Msg {
		count := token.CountTokens(promptMsgs)
		return tokenCountResultMsg{sessID: sessID, count: count}
	}
}

func (m *Model) addActiveSession(sess *coder.Session) {
	for i, s := range m.ActiveSessions {
		if s.ID == sess.ID {
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
		if sess.ID == id {
			return func() tea.Msg { return switchActiveSessionMsg{sess: sess} }
		}
	}
	return nil
}

func (m Model) getSessionByID(id string) *coder.Session {
	if id == "" {
		return m.Session
	}
	if m.Session != nil && m.Session.ID == id {
		return m.Session
	}
	for _, s := range m.ActiveSessions {
		if s.ID == id {
			return s
		}
	}
	return nil
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
