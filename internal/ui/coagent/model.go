package coagentui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/coagent"
	"github.com/sokinpui/coder/internal/types"
)

type state int

const (
	stateInput state = iota
	stateRunning
)

type streamChunkMsg coagent.AgentStreamChunk
type agentFinishedMsg struct{}
type agentErrorMsg struct{ err error }

type Model struct {
	state          state
	runtime        *coagent.AgentRuntime
	cancelFunc     context.CancelFunc
	input          textinput.Model
	spinner        spinner.Model
	messages       []types.Message
	chunkChan      chan coagent.AgentStreamChunk
	currentContent strings.Builder
	initialPrompt  string
	startedInitial bool
}

func New(cfg *config.Config, initialPrompt string) (Model, error) {
	runtime, err := coagent.NewAgentRuntime(cfg, nil)
	if err != nil {
		return Model{}, err
	}

	ti := textinput.New()
	ti.Placeholder = "Ask agent anything (Ctrl+C to quit)..."
	ti.Prompt = ""
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	return Model{
		state:         stateInput,
		runtime:       runtime,
		input:         ti,
		spinner:       sp,
		initialPrompt: strings.TrimSpace(initialPrompt),
	}, nil
}

func (m Model) Init() tea.Cmd {
	if m.initialPrompt != "" && !m.startedInitial {
		return m.runPrompt(m.initialPrompt)
	}
	return textinput.Blink
}
