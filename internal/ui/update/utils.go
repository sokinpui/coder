package update

import (
	"coder/internal/history"
	"coder/internal/session"
	"coder/internal/token"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listenForStream waits for the next message from the generation stream.
func listenForStream(sub chan string) tea.Cmd {
	return func() tea.Msg {
		content, ok := <-sub
		if !ok {
			return streamFinishedMsg{}
		}
		if strings.HasPrefix(content, "Error:") {
			errMsg := strings.TrimSpace(strings.TrimPrefix(content, "Error:"))
			return errorMsg{errors.New(errMsg)}
		}
		return streamResultMsg(content)
	}
}

func countTokensCmd(text string) tea.Cmd {
	return func() tea.Msg {
		count := token.CountTokens(text)
		return tokenCountResultMsg(count)
	}
}

func loadInitialContextCmd(sess *session.Session) tea.Cmd {
	return func() tea.Msg {
		err := sess.LoadContext()
		return initialContextLoadedMsg{err: err}
	}
}

func listHistoryCmd(histMgr *history.Manager) tea.Cmd {
	return func() tea.Msg {
		items, err := histMgr.ListConversations()
		return historyListResultMsg{items: items, err: err}
	}
}

func loadConversationCmd(sess *session.Session, filename string) tea.Cmd {
	return func() tea.Msg {
		err := sess.LoadConversation(filename)
		return conversationLoadedMsg{err: err}
	}
}

func renderTick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return renderTickMsg{}
	})
}

func animateTitleTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return animateTitleTickMsg{}
	})
}

func ctrlCTimeout() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return ctrlCTimeoutMsg{}
	})
}

func clearStatusBarCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearStatusBarMsg{}
	})
}

func generateTitleCmd(sess *session.Session, userPrompt string) tea.Cmd {
	return func() tea.Msg {
		// This runs in a goroutine managed by Bubble Tea.
		title := sess.GenerateTitle(context.Background(), userPrompt)
		return titleGeneratedMsg{title: title}
	}
}

func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	return editor
}

func editInEditorCmd(content string) tea.Cmd {
	editor := getEditor()
	tmpfile, err := os.CreateTemp("", "coder-*.md")
	if err != nil {
		return func() tea.Msg { return errorMsg{err} }
	}

	if _, err := tmpfile.WriteString(content); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return func() tea.Msg { return errorMsg{err} }
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return func() tea.Msg { return errorMsg{err} }
	}

	cmd := exec.Command(editor, tmpfile.Name())

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmpfile.Name())
		if err != nil {
			return editorFinishedMsg{err: err}
		}

		newContent, readErr := os.ReadFile(tmpfile.Name())
		if readErr != nil {
			return editorFinishedMsg{err: readErr}
		}

		return editorFinishedMsg{content: string(newContent)}
	})
}

// calculateVisibleLines calculates the number of lines a text block will occupy
// in the textarea, considering word wrapping.
func calculateVisibleLines(text string, width int) int {
	if width <= 0 {
		// Avoid division by zero and handle cases where width is not yet set.
		return 1
	}
	if text == "" {
		return 1
	}

	lines := strings.Split(text, "\n")
	visibleLineCount := 0
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth == 0 {
			visibleLineCount++ // Empty line still takes up one line.
		} else {
			// Integer division to calculate wrapped lines.
			visibleLineCount += (lineWidth-1)/width + 1
		}
	}
	return visibleLineCount
}
