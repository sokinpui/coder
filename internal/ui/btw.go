package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/coder/internal/clipboard"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
)

type BtwModel struct {
	Active              bool
	RestoreAfterConfirm bool
	Snapshot            []types.Message
	Messages            []types.Message
	Input               textarea.Model
	Viewport            viewport.Model
	StreamSub           <-chan types.StreamChunk
	CancelFunc          context.CancelFunc
	IsStreaming         bool
	StatusText          string
	Width               int
	Height              int
	RenderCache         map[int]markdown.CachedRender
	AutoScroll          bool
}

func NewBtwModel() BtwModel {
	ta := textarea.New()
	ta.Placeholder = "Ask a side question... (Enter / Ctrl+J: send, Esc: close)"
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	return BtwModel{
		Input:       ta,
		Viewport:    viewport.New(60, 12),
		RenderCache: make(map[int]markdown.CachedRender),
		AutoScroll:  true,
	}
}

func (b *BtwModel) reset() {
	if b.CancelFunc != nil {
		b.CancelFunc()
	}
	*b = NewBtwModel()
}

func (b *BtwModel) layout(totalWidth, totalHeight int) {
	maxWidth := max(40, totalWidth-4)
	maxHeight := max(12, totalHeight-4)
	modalWidth := max(50, min((totalWidth*4)/5, maxWidth))
	modalHeight := max(14, min((totalHeight*4)/5, maxHeight))

	b.Width = modalWidth
	b.Height = modalHeight

	innerWidth := max(20, modalWidth-PaletteContainerStyle.GetHorizontalFrameSize())
	innerHeight := max(6, modalHeight-PaletteContainerStyle.GetVerticalFrameSize())

	inputInnerWidth := max(16, innerWidth-TextAreaContainerStyle.GetHorizontalFrameSize())
	b.Input.SetWidth(inputInnerWidth)
	visibleLines := CalculateVisibleLines(b.Input, inputInnerWidth, 4)
	inputHeight := max(1, min(visibleLines, 3))
	b.Input.SetHeight(inputHeight)

	inputBoxHeight := inputHeight + TextAreaContainerStyle.GetVerticalFrameSize()
	headerHeight := 2
	statusHeight := 1
	vpHeight := max(3, innerHeight-headerHeight-inputBoxHeight-statusHeight)

	b.Viewport.Width = innerWidth
	b.Viewport.Height = vpHeight
}

func (b *BtwModel) renderConversation(spinnerView string) string {
	if len(b.Messages) == 0 {
		return ToolMutedStyle.Render("Ask a side question using current context without interrupting or saving to history.")
	}

	width := max(20, b.Viewport.Width)
	var parts []string
	for i, msg := range b.Messages {
		switch msg.Type {
		case types.UserMessage:
			rendered := UserInputStyle.Width(width - UserInputStyle.GetHorizontalFrameSize()).Render(msg.Content)
			parts = append(parts, rendered)
		case types.CommandErrorResultMessage:
			rendered := CommandErrorStyle.Width(width - CommandErrorStyle.GetHorizontalFrameSize()).Render(msg.Content)
			parts = append(parts, rendered)
		case types.AIMessage:
			if msg.Content == "" {
				continue
			}
			if b.IsStreaming && i == len(b.Messages)-1 {
				parts = append(parts, lipgloss.NewStyle().Width(width).Render(msg.Content))
				continue
			}
			if cached, ok := b.RenderCache[i]; ok && cached.Content == msg.Content && cached.Width == width {
				parts = append(parts, strings.Join(cached.Lines, "\n"))
				continue
			}
			rendered := markdown.Render(msg.Content, width)
			lines := strings.Split(strings.ReplaceAll(rendered, "\r\n", "\n"), "\n")
			b.RenderCache[i] = markdown.CachedRender{
				Lines:   lines,
				Content: msg.Content,
				Width:   width,
			}
			parts = append(parts, strings.Join(lines, "\n"))
		}
	}

	if b.IsStreaming {
		status := "Thinking "
		if b.StatusText != "" {
			status = b.StatusText + " "
		}
		parts = append(parts, RenderThinkingSpinner(status, spinnerView))
	}

	return strings.Join(parts, "\n\n")
}

func (b *BtwModel) refreshViewport(spinnerView string) {
	b.Viewport.SetContent(b.renderConversation(spinnerView))
	if b.AutoScroll {
		b.Viewport.GotoBottom()
	}
}

type BtwOverlay struct{}

func (o *BtwOverlay) IsVisible(main *Model) bool {
	return main.ActiveOverlay == overlayBtw && main.Btw.Active
}

func (o *BtwOverlay) View(main *Model) string {
	main.Btw.layout(main.Width, main.Height)
	main.Btw.refreshViewport(main.Chat.Spinner.View())

	header := PaletteHeaderStyle.Render("Side Question (/btw) [Esc: close | Enter/Ctrl+J: ask | Ctrl+U/D: scroll]")
	inputBox := TextAreaContainerStyle.Render(main.Btw.Input.View())

	var statusLine string
	if main.Btw.IsStreaming {
		status := "Thinking "
		if main.Btw.StatusText != "" {
			status = main.Btw.StatusText + " "
		}
		statusLine = lipgloss.JoinHorizontal(
			lipgloss.Bottom,
			ThinkingStatusStyle.Render(status),
			main.Chat.Spinner.View(),
			ToolMutedStyle.Render("  [Ctrl+C: stop]"),
		)
	} else {
		statusLine = ToolMutedStyle.Render("Ephemeral side chat (not saved to session history)")
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		main.Btw.Viewport.View(),
		inputBox,
		statusLine,
	)

	modal := PaletteContainerStyle.Width(main.Btw.Width).Render(body)
	return OverlayCenter(modal, main.View())
}

func (m Model) openBtwOverlay(initialQuestion string) (Model, tea.Cmd, bool) {
	sq, ok := m.Session.(engine.SideQuestioner)
	if !ok {
		m.StatusBarMessage = "Side questions (/btw) are not supported in this session."
		return m, clearStatusBarCmd(), true
	}

	snapshot, err := sq.SnapshotMessages()
	if err != nil {
		m.StatusBarMessage = fmt.Sprintf("Failed to snapshot context for /btw: %v", err)
		return m, clearStatusBarCmd(), true
	}

	m.Btw.reset()
	m.Btw.Active = true
	m.Btw.Snapshot = snapshot
	m.Btw.layout(m.Width, m.Height)
	m.Btw.Input.Focus()
	m.Btw.refreshViewport(m.Chat.Spinner.View())

	m.ActiveOverlay = overlayBtw
	m.Chat.ShowPalette = false
	m.Chat.TextArea.Blur()

	q := strings.TrimSpace(initialQuestion)
	if q == "" {
		return m, textarea.Blink, true
	}

	newModel, cmd := m.submitBtwQuestion(q)
	return newModel, cmd, true
}

func (m Model) submitBtwQuestion(question string) (Model, tea.Cmd) {
	sq, ok := m.Session.(engine.SideQuestioner)
	if !ok {
		return m, nil
	}

	m.Btw.Messages = append(m.Btw.Messages, types.Message{
		Type:    types.UserMessage,
		Content: question,
	})

	reqMessages := make([]types.Message, 0, len(m.Btw.Snapshot)+len(m.Btw.Messages))
	reqMessages = append(reqMessages, m.Btw.Snapshot...)
	reqMessages = append(reqMessages, m.Btw.Messages...)

	if m.Btw.CancelFunc != nil {
		m.Btw.CancelFunc()
	}
	ctx, cancel := context.WithCancel(context.Background())

	sub, err := sq.AskSideQuestion(ctx, reqMessages)
	if err != nil {
		cancel()
		m.Btw.Messages = append(m.Btw.Messages, types.Message{
			Type:    types.CommandErrorResultMessage,
			Content: err.Error(),
		})
		m.Btw.AutoScroll = true
		m.Btw.refreshViewport(m.Chat.Spinner.View())
		return m, textarea.Blink
	}

	m.Btw.CancelFunc = cancel
	m.Btw.StreamSub = sub
	m.Btw.IsStreaming = true
	m.Btw.StatusText = "Asking"
	m.Btw.AutoScroll = true
	m.Btw.refreshViewport(m.Chat.Spinner.View())

	return m, tea.Batch(
		listenForBtwChunks(sub),
		m.Chat.Spinner.Tick,
		textarea.Blink,
	)
}

func (m Model) closeBtwOverlay() (Model, tea.Cmd) {
	m.Btw.reset()
	m.ActiveOverlay = overlayNone
	if m.shouldFocusTextArea() {
		m.Chat.TextArea.Focus()
		return m, textarea.Blink
	}
	return m, nil
}

func (m Model) handleKeyPressBtw(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	km := m.Keymap()
	keyStr := msg.String()

	switch msg.Type {
	case tea.KeyEsc:
		newModel, cmd := m.closeBtwOverlay()
		return newModel, cmd, true

	case tea.KeyCtrlC:
		if m.Btw.IsStreaming {
			if m.Btw.CancelFunc != nil {
				m.Btw.CancelFunc()
				m.Btw.CancelFunc = nil
			}
			m.Btw.StreamSub = nil
			m.Btw.IsStreaming = false
			m.Btw.StatusText = ""
			m.Btw.Messages = append(m.Btw.Messages, types.Message{
				Type:    types.CommandErrorResultMessage,
				Content: "Interrupted by user.",
			})
			m.Btw.refreshViewport(m.Chat.Spinner.View())
			return m, textarea.Blink, true
		}
		newModel, cmd := m.closeBtwOverlay()
		return newModel, cmd, true

	case tea.KeyPgUp:
		m.Btw.AutoScroll = false
		m.Btw.Viewport.PageUp()
		return m, nil, true

	case tea.KeyPgDown:
		m.Btw.Viewport.PageDown()
		m.Btw.AutoScroll = m.Btw.Viewport.AtBottom()
		return m, nil, true
	}

	switch keyStr {
	case km.ScrollUp:
		m.Btw.AutoScroll = false
		m.Btw.Viewport.HalfPageUp()
		return m, nil, true

	case km.ScrollDown:
		m.Btw.Viewport.HalfPageDown()
		m.Btw.AutoScroll = m.Btw.Viewport.AtBottom()
		return m, nil, true

	case km.Paste:
		return m, btwPasteCmd(), true
	}

	if (msg.Type == tea.KeyEnter && !msg.Alt) || keyStr == km.Submit {
		if m.Btw.IsStreaming {
			return m, nil, true
		}
		q := strings.TrimSpace(m.Btw.Input.Value())
		if q == "" {
			return m, nil, true
		}
		m.Btw.Input.Reset()
		m.Btw.layout(m.Width, m.Height)
		newModel, cmd := m.submitBtwQuestion(q)
		return newModel, cmd, true
	}

	var cmd tea.Cmd
	m.Btw.Input, cmd = m.Btw.Input.Update(msg)
	if cleaned := CleanSGRLeakedText(m.Btw.Input.Value()); cleaned != m.Btw.Input.Value() {
		m.Btw.Input.SetValue(cleaned)
	}
	m.Btw.layout(m.Width, m.Height)
	return m, cmd, true
}

func listenForBtwChunks(sub <-chan types.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-sub
		if !ok {
			return btwStreamFinishedMsg{sub: sub}
		}
		return btwStreamChunkMsg{
			sub:   sub,
			chunk: chunk,
		}
	}
}

func btwPasteCmd() tea.Cmd {
	return func() tea.Msg {
		text, err := clipboard.PasteText()
		return btwPasteResultMsg{content: text, err: err}
	}
}
