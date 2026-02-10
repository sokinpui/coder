package ui

import (
	"context"
	"errors"
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/history"
	"github.com/sokinpui/coder/internal/session"
	"github.com/sokinpui/coder/internal/token"
	"github.com/sokinpui/coder/internal/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sokinpui/synapse.go/client"
)

const statusBarMessageDuration = 1 * time.Second

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

func fetchModelsCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		c, err := client.New(cfg.GRPC.Addr)
		if err != nil {
			return modelsFetchedMsg{err: fmt.Errorf("error connecting to server: %w", err)}
		}
		defer c.Close()

		models, err := c.ListModels(context.Background())
		if err != nil {
			return modelsFetchedMsg{err: fmt.Errorf("error fetching models from server: %w", err)}
		}

		return modelsFetchedMsg{models: models}
	}
}

func initTokenizerCmd() tea.Cmd {
	return func() tea.Msg {
		err := token.Init()
		return tokenizerInitializedMsg{err: err}
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

func saveConversationCmd(sess *session.Session) tea.Cmd {
	return func() tea.Msg {
		if err := sess.SaveConversation(); err != nil {
			return errorMsg{err}
		}
		return nil
	}
}

func streamAnimeCmd(delay int) tea.Cmd {
	return tea.Tick(time.Duration(delay)*time.Millisecond, func(t time.Time) tea.Msg {
		return streamAnimeMsg{}
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

func clearStatusBarCmd() tea.Cmd {
	return tea.Tick(statusBarMessageDuration, func(t time.Time) tea.Msg {
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

// getVisibleLines calculates the number of lines a text block will occupy
// in the textarea, considering word wrapping.
func getVisibleLines(ta textarea.Model, width int, maxLines int) int {
	if width <= 0 {
		// Avoid division by zero and handle cases where width is not yet set.
		return 1
	}

	visibleLineCount := 0
	for line := range strings.SplitSeq(ta.Value(), "\n") {
		lineWidth := lipgloss.Width(line)
		if lineWidth == 0 {
			visibleLineCount++ // Empty line still takes up one line.
		} else {
			// Integer division to calculate wrapped lines.
			visibleLineCount += (lineWidth-1)/width + 1
		}
		if maxLines > 0 && visibleLineCount > maxLines {
			return visibleLineCount
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

// handlePasteCmd checks the clipboard for an image.
// It uses `pngpaste` on macOS and `xclip` on Linux.
// If an image is found, it's saved to `.coder/images` and the relative path is returned.
// If not, it falls back to pasting text content.
func handlePasteCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if cfg.Clipboard.PasteCmd != "" {
			data, contentType, err := utils.PasteCustom(cfg.Clipboard.PasteCmd)
			if err != nil {
				return pasteResultMsg{err: err}
			}

			if !strings.HasPrefix(contentType, "image/") {
				return pasteResultMsg{isImage: false, content: string(data)}
			}

			ext := ".png"
			switch contentType {
			case "image/jpeg":
				ext = ".jpg"
			case "image/webp":
				ext = ".webp"
			}

			relPath, err := saveImageToRepo(data, ext)
			if err != nil {
				return pasteResultMsg{err: err}
			}
			return pasteResultMsg{isImage: true, content: relPath}
		}

		if data, _, err := utils.GetImageFromClipboard(); err == nil {
			if relPath, err := saveImageToRepo(data, ".png"); err == nil {
				return pasteResultMsg{isImage: true, content: relPath}
			}
		}

		content, err := utils.PasteText()
		if err != nil {
			return pasteResultMsg{err: fmt.Errorf("failed to read clipboard: %w", err)}
		}
		return pasteResultMsg{isImage: false, content: content}
	}
}

func saveImageToRepo(data []byte, ext string) (string, error) {
	repoRoot := utils.GetProjectRoot()
	imagesDir := filepath.Join(repoRoot, ".coder", "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return "", fmt.Errorf("could not create images directory: %w", err)
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(imagesDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save image file: %w", err)
	}

	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return filePath, nil
	}
	return filepath.ToSlash(relPath), nil
}
