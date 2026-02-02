package ui

import (
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
	QuickView    *QuickViewModel

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

func NewModel(cfg *config.Config, mode string, initialInput string) (Model, error) {
	sess, err := session.New(cfg, mode)
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
		Chat:              NewChat(initialInput),
		VisualSelect:      NewVisualSelect(),
		History:           NewHistory(),
		Search:            NewSearch(),
		Finder:            NewFinder(),
		Tree:              NewTree(),
		QuickView:         NewQuickView(),
		Session:           sess,
		State:             stateInitializing,
		GlamourRenderer:   renderer,
		AvailableCommands: availableCommands,
	}, nil
}
