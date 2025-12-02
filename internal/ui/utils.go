package ui

import (
	"coder/internal/config"
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
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
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

// handlePasteCmd checks the clipboard for an image.
// It uses `pngpaste` on macOS and `xclip` on Linux.
// If an image is found, it's saved to `.coder/images` and the relative path is returned.
// If not, it falls back to pasting text content.
func handlePasteCmd() tea.Cmd {
	return func() tea.Msg {
		// Try to paste an image based on OS
		switch runtime.GOOS {
		case "darwin":
			if _, err := exec.LookPath("pngpaste"); err != nil {
				break // pngpaste not found, fallback to text
			}

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
			if err := cmd.Run(); err == nil {
				// Success! It was an image. Return relative path
				relPath, err := filepath.Rel(repoRoot, filePath)
				if err != nil { // Should not happen, but fallback to full path
					relPath = filePath
				}
				return pasteResultMsg{isImage: true, content: filepath.ToSlash(relPath)}
			}
			// If pngpaste fails, it means no image was on clipboard. Fall through to text paste.

		case "linux":
			isWayland := os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"

			var checkCmd, saveCmd *exec.Cmd

			if isWayland {
				if _, err := exec.LookPath("wl-paste"); err != nil {
					break // wl-paste not found, fallback to text
				}
				checkCmd = exec.Command("wl-paste", "--list-types")
				saveCmd = exec.Command("wl-paste", "-t", "image/png")
			} else {
				if _, err := exec.LookPath("xclip"); err != nil {
					break // xclip not found, fallback to text
				}
				checkCmd = exec.Command("xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
				saveCmd = exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
			}

			// Check if clipboard has image/png target
			output, err := checkCmd.Output()
			if err != nil || !strings.Contains(string(output), "image/png") {
				break // No PNG image on clipboard, fallback to text
			}

			// Save the image
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

			outFile, err := os.Create(filePath)
			if err != nil {
				return pasteResultMsg{err: fmt.Errorf("could not create image file: %w", err)}
			}
			defer outFile.Close()

			saveCmd.Stdout = outFile

			if err := saveCmd.Run(); err == nil {
				info, statErr := os.Stat(filePath)
				if statErr == nil && info.Size() > 0 {
					// Success! It was an image. Return relative path
					relPath, err := filepath.Rel(repoRoot, filePath)
					if err != nil { // Should not happen, but fallback to full path
						relPath = filePath
					}
					return pasteResultMsg{isImage: true, content: filepath.ToSlash(relPath)}
				}
				os.Remove(filePath) // clean up empty file
			}
			// If tool fails, fall through to text paste.
		}

		// Fallback for other OSes or if image paste tool fails/is not present
		content, err := clipboard.ReadAll()
		if err != nil {
			return pasteResultMsg{err: fmt.Errorf("failed to read clipboard: %w", err)}
		}
		return pasteResultMsg{isImage: false, content: content}
	}
}
