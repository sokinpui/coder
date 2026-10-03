package coagentui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/engine/token"
	coderui "github.com/sokinpui/coder/internal/ui/coder"
	"github.com/sokinpui/coder/internal/ui/core"
)

type state int

const (
	stateInput state = iota
	stateThinking
	stateGenerating
)

type overlayMode int

const (
	overlayNone overlayMode = iota
	overlaySelector
)

type streamChunkMsg struct {
	sessID string
	chunk  coagent.AgentStreamChunk
}
type agentFinishedMsg struct {
	sessID string
}
type agentErrorMsg struct {
	sessID string
	err    error
}
type initPromptMsg string

type Model struct {
	cfg              *config.Config
	state            state
	activeSessions   []*coagent.Session
	session          *coagent.Session
	cancelFunc       context.CancelFunc
	activeOverlay    overlayMode
	input            core.InputBox
	viewport         core.Viewport
	spinner          spinner.Model
	chunkChan        chan coagent.AgentStreamChunk
	initialPrompt    string
	statusText       string
	stateStart       time.Time
	width            int
	height           int
	ready            bool
	ctrlCPressed     bool
	tokenCount       int
	editingMsgIdx    int
	statusBarMessage string
	selector         coderui.SelectorModel
	showSelector     bool
	animatingTitle   bool
	fullTitle        string
	displayTitle     string
	isAIRendering    bool
	pendingAIRender  bool
	toolsExpanded    bool
}

func New(cfg *config.Config, initialPrompt string, instruction ...string) (Model, error) {
	sess, err := coagent.NewSession(cfg)
	if err != nil {
		return Model{}, err
	}

	if len(instruction) > 0 && instruction[0] != "" {
		sess.Instruction = instruction[0]
	}

	ib := core.NewInputBox("Ask agent anything (Ctrl+C to quit)...")

	sp := spinner.New()
	sp.Spinner = core.TypingSpinner
	sp.Style = core.GeneratingStatusStyle

	vp := core.NewViewport(80, 20)

	m := Model{
		cfg:            cfg,
		state:          stateInput,
		activeSessions: []*coagent.Session{sess},
		session:        sess,
		input:          ib,
		viewport:       vp,
		spinner:        sp,
		initialPrompt:  strings.TrimSpace(initialPrompt),
		width:          80,
		height:         24,
		ready:          true,
		selector:       coderui.NewSelector(),
		editingMsgIdx:  -1,
		tokenCount:     token.CountTokens(sess.GetPrompt()),
	}
	return m.updateLayout(), nil
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, m.renderUncachedCmd()}
	if m.initialPrompt != "" {
		cmds = append(cmds, func() tea.Msg { return initPromptMsg(m.initialPrompt) })
	}
	return tea.Batch(cmds...)
}

func (m Model) renderUncachedCmd() tea.Cmd {
	if m.session == nil {
		return nil
	}
	isStreaming := m.state == stateGenerating || m.state == stateThinking
	return m.viewport.RenderUncachedCmd(m.session.ID, m.session.Messages, isStreaming)
}

func (m *Model) addActiveSession(sess *coagent.Session) {
	for i, s := range m.activeSessions {
		if s.ID == sess.ID {
			m.activeSessions[i] = sess
			return
		}
		if sess.HistoryFilename != "" && s.HistoryFilename == sess.HistoryFilename {
			m.activeSessions[i] = sess
			return
		}
	}
	m.activeSessions = append(m.activeSessions, sess)
}

func (m Model) getSessionByID(id string) *coagent.Session {
	if id == "" || (m.session != nil && m.session.ID == id) {
		return m.session
	}
	for _, s := range m.activeSessions {
		if s.ID == id {
			return s
		}
	}
	return nil
}
