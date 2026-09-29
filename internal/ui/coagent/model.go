package coagentui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/core"
)

type state int

const (
	stateInput state = iota
	stateThinking
	stateGenerating
)

type streamChunkMsg coagent.AgentStreamChunk
type agentFinishedMsg struct{}
type agentErrorMsg struct{ err error }
type initPromptMsg string

type Model struct {
	cfg           *config.Config
	state         state
	runtime       *coagent.AgentRuntime
	cancelFunc    context.CancelFunc
	input         core.InputBox
	viewport      core.Viewport
	spinner       spinner.Model
	messages      []types.Message
	chunkChan     chan coagent.AgentStreamChunk
	initialPrompt string
	statusText    string
	stateStart    time.Time
	width         int
	height        int
	ready         bool
	ctrlCPressed  bool
	tokenCount    int
}

func New(cfg *config.Config, initialPrompt string) (Model, error) {
	runtime, err := coagent.NewAgentRuntime(cfg, nil)
	if err != nil {
		return Model{}, err
	}

	ib := core.NewInputBox("Ask agent anything (Ctrl+C to quit)...")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = core.GeneratingStatusStyle

	vp := core.NewViewport(80, 20)

	m := Model{
		cfg:           cfg,
		state:         stateInput,
		runtime:       runtime,
		input:         ib,
		viewport:      vp,
		spinner:       sp,
		initialPrompt: strings.TrimSpace(initialPrompt),
		width:         80,
		height:        24,
		ready:         true,
	}
	return m.updateLayout(), nil
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}
	if m.initialPrompt != "" {
		cmds = append(cmds, func() tea.Msg { return initPromptMsg(m.initialPrompt) })
	}
	return tea.Batch(cmds...)
}
