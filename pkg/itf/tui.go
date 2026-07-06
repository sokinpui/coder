package itf

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	createdStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	renamedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	deletedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("197"))
)

type spinner struct {
	frames []string
	index  int
}

func newSpinner() spinner { return spinner{frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}} }
func (s *spinner) tick()   { s.index = (s.index + 1) % len(s.frames) }
func (s spinner) View() string { return s.frames[s.index] }

type TUI struct {
	app         *App
	noAnimation bool
	spinner     spinner
	mu          sync.Mutex
	cur, total  int
}

func NewTUI(app *App, noAnimation bool) *TUI {
	return &TUI{app: app, noAnimation: noAnimation, spinner: newSpinner()}
}

func (t *TUI) Run() error {
	if t.noAnimation {
		summary, err := t.app.Execute()
		if err == nil {
			fmt.Print(FormatSummary(summary))
		}
		return err
	}

	t.app.SetProgressCallback(func(c, tot int) {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.cur, t.total = c, tot
	})

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(100 * time.Millisecond):
				t.spinner.tick()
				t.renderProgress()
			}
		}
	}()

	summary, err := t.app.Execute()
	close(done)
	fmt.Print("\r\x1b[K")

	if err == nil {
		fmt.Print(FormatSummary(summary))
	}
	return err
}

func (t *TUI) renderProgress() {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Printf("\r%s Processing... %d/%d\x1b[K", t.spinner.View(), t.cur, t.total)
}

func FormatSummary(s Summary) string {
	var b strings.Builder
	if s.Message != "" {
		b.WriteString(headerStyle.Render(s.Message) + "\n\n")
	}

	renderList := func(title string, style lipgloss.Style, list []string) {
		if len(list) == 0 {
			return
		}
		b.WriteString(style.Render(title) + "\n")
		for _, f := range list {
			b.WriteString(fmt.Sprintf("  %s\n", f))
		}
	}

	renderList("Created:", createdStyle, s.Created)
	renderList("Modified:", successStyle, s.Modified)
	renderList("Renamed:", renamedStyle, s.Renamed)
	renderList("Deleted:", deletedStyle, s.Deleted)
	renderList("Failed:", errorStyle, s.Failed)

	return b.String()
}
