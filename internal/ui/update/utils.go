package update

import (
	"coder/internal/history"
	"coder/internal/session"
	"coder/internal/token"
	"coder/internal/utils"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
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
		if errMsg, result := strings.CutPrefix(content, "Error:"); result {
			return errorMsg{errors.New(strings.TrimSpace(errMsg))}
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
			return editorFinishedMsg{err: err, originalContent: content}
		}

		newContent, readErr := os.ReadFile(tmpfile.Name())
		if readErr != nil {
			return editorFinishedMsg{err: readErr, originalContent: content}
		}

		return editorFinishedMsg{content: string(newContent), originalContent: content}
	})
}

func runFzfCmd(input string) tea.Cmd {
	tmpfile, err := os.CreateTemp("", "coder-fzf-result-")
	if err != nil {
		return func() tea.Msg { return fzfFinishedMsg{err: err} }
	}
	tmpfileName := tmpfile.Name()
	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfileName)
		return func() tea.Msg { return fzfFinishedMsg{err: err} }
	}

	// Use sh -c to handle redirection, more portable than bash
	var fzfCmdStr string
	if os.Getenv("TMUX") != "" {
		fzfCmdStr = fmt.Sprintf("fzf-tmux -p 100%%,100%% > %s", tmpfileName)
	} else {
		fzfCmdStr = fmt.Sprintf("fzf > %s", tmpfileName)
	}

	cmd := exec.Command("sh", "-c", fzfCmdStr)
	cmd.Stdin = strings.NewReader(input)

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmpfileName)

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
				return fzfFinishedMsg{result: "", err: nil} // User cancelled
			}
			return fzfFinishedMsg{err: err}
		}

		resultBytes, readErr := os.ReadFile(tmpfileName)
		if readErr != nil {
			return fzfFinishedMsg{err: readErr}
		}

		return fzfFinishedMsg{result: strings.TrimSpace(string(resultBytes))}
	})
}

// getVisibleLines calculates the number of lines a text block will occupy
// in the textarea, considering word wrapping.
func getVisibleLines(text string, width int) int {
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

// cursorPosAfterScroll calculates the new cursor position after a half-page scroll.
func cursorPosAfterScroll(currentCursor, scrollAmount, totalItems int, scrollDown bool) int {
	if totalItems == 0 {
		return 0
	}

	var newCursor int
	if scrollDown {
		newCursor = currentCursor + scrollAmount
		if newCursor >= totalItems {
			newCursor = totalItems - 1
		}
	} else { // scroll up
		newCursor = currentCursor - scrollAmount
		if newCursor < 0 {
			newCursor = 0
		}
	}
	return newCursor
}

// handlePasteCmd checks the clipboard for an image using `pngpaste`.
// If an image is found, it's saved to `.coder/images` and the relative path is returned.
// If not, it falls back to pasting text content.
func handlePasteCmd() tea.Cmd {
	return func() tea.Msg {
		// 1. Check if pngpaste exists
		if _, err := exec.LookPath("pngpaste"); err != nil {
			// pngpaste not found, fallback to text paste
			content, err := clipboard.ReadAll()
			if err != nil {
				return pasteResultMsg{err: fmt.Errorf("failed to read clipboard: %w", err)}
			}
			return pasteResultMsg{isImage: false, content: content}
		}

		// 2. Check if clipboard has image data by trying to save it.
		repoRoot, err := utils.FindRepoRoot()
		if err != nil {
			return pasteResultMsg{err: fmt.Errorf("could not find repo root: %w", err)}
		}

		imagesDir := filepath.Join(repoRoot, ".coder", "images")
		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			return pasteResultMsg{err: fmt.Errorf("could not create images directory: %w", err)}
		}

		filename := fmt.Sprintf("%d.png", time.Now().UnixNano())
		filePath := filepath.Join(imagesDir, filename)

		cmd := exec.Command("pngpaste", filePath)
		if err := cmd.Run(); err != nil {
			// This error means there was no image on the clipboard. Fallback to text paste.
			content, readErr := clipboard.ReadAll()
			if readErr != nil {
				return pasteResultMsg{err: fmt.Errorf("pngpaste failed and could not read clipboard text: %w", readErr)}
			}
			return pasteResultMsg{isImage: false, content: content}
		}

		// Success! It was an image. Return relative path
		relPath, err := filepath.Rel(repoRoot, filePath)
		if err != nil { // Should not happen, but fallback to full path
			relPath = filePath
		}
		return pasteResultMsg{isImage: true, content: filepath.ToSlash(relPath)}
	}
}
