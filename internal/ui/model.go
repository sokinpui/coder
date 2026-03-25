package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/commands"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"sort"

	"github.com/charmbracelet/glamour"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Model struct {
	Chat         ChatModel
	VisualSelect VisualSelectModel
	History      HistoryModel
	Finder       FinderModel
	Search       SearchModel
	Tree         TreeModel
	Jump         JumpModel
	QuickView    *QuickViewModel

	ActiveSessions    []*session.Session
	Session           *session.Session
	State             state
	Quitting          bool
	Height            int
	Width             int
	GlamourRenderer   *glamour.TermRenderer
	AvailableCommands []string
	StatusBarMessage  string
	TokenCount        int
	IsCountingTokens  bool
}

func NewModel(cfg *config.Config, mode string, initialInput string, contextFiles []string, instruction string) (Model, error) {
	sess, err := session.New(cfg, mode, instruction, contextFiles)
	if err != nil {
		return Model{}, err
	}
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle(cfg.UI.MarkdownTheme),
		glamour.WithWordWrap(80),
	)

	sess.AddMessages(types.Message{Type: types.InitMessage, Content: utils.WelcomeMessage})

	dirMsg := utils.GetDirInfoContent()
	sess.AddMessages(types.Message{Type: types.DirectoryMessage, Content: dirMsg})
	availableCommands := commands.GetCommands()
	sort.Strings(availableCommands)

	return Model{
		ActiveSessions:    []*session.Session{sess},
		Chat:              NewChat(initialInput),
		VisualSelect:      NewVisualSelect(),
		History:           NewHistory(),
		Search:            NewSearch(),
		Finder:            NewFinder(),
		Tree:              NewTree(),
		Jump:              NewJump(),
		QuickView:         NewQuickView(),
		Session:           sess,
		State:             stateInitializing,
		GlamourRenderer:   renderer,
		AvailableCommands: availableCommands,
	}, nil
}

func (m *Model) addActiveSession(sess *session.Session) {
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

func (m Model) determineStreamingState() state {
	if !m.Chat.IsStreaming {
		return stateIdle
	}

	messages := m.Session.GetMessages()
	if len(messages) == 0 {
		return stateQueuing
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Type != types.AIMessage {
		return stateQueuing
	}

	if lastMsg.Content != "" {
		return stateGenerating
	}

	if m.Chat.ThoughtBuffer != "" {
		return stateThinking
	}

	return stateQueuing
}
