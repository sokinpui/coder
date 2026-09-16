package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sokinpui/coder/internal/clipboard"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/session"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/pkg/sf"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const statusBarMessageDuration = 1 * time.Second

func listenForStream(sessID string, sub chan types.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-sub
		if !ok {
			return streamFinishedMsg{sessID: sessID}
		}
		if errMsg, result := strings.CutPrefix(chunk.Content, "Error:"); result {
			return errorMsg{sessID: sessID, error: errors.New(strings.TrimSpace(errMsg))}
		}
		return streamResultMsg{
			sessID: sessID,
			chunk:  chunk,
			sub:    sub,
		}
	}
}

func renderAIMessageCmd(sessID string, msgIdx int, content string, width int, theme string) tea.Cmd {
	return func() tea.Msg {
		if content == "" {
			return aiRenderedMsg{
				sessID:  sessID,
				msgIdx:  msgIdx,
				content: content,
				lines:   nil,
				width:   width,
			}
		}

		renderer, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle(theme),
			glamour.WithWordWrap(width),
		)
		var rendered string
		if err == nil {
			rendered, err = renderer.Render(content)
		}
		if err != nil {
			rendered = content
		}
		return aiRenderedMsg{
			sessID:  sessID,
			msgIdx:  msgIdx,
			content: content,
			lines:   strings.Split(rendered, "\n"),
			width:   width,
		}
	}
}

func fetchModelsCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		endpoint := strings.TrimSuffix(cfg.Server.URL, "/") + "/models"

		resp, err := http.Get(endpoint)
		if err != nil {
			return modelsFetchedMsg{err: fmt.Errorf("failed to reach %s: %w", endpoint, err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return modelsFetchedMsg{err: fmt.Errorf("server returned status %d", resp.StatusCode)}
		}

		type openAIModel struct {
			ID string `json:"id"`
		}
		type openAIModelList struct {
			Data []openAIModel `json:"data"`
		}

		var result openAIModelList
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return modelsFetchedMsg{err: fmt.Errorf("error decoding models: %w", err)}
		}

		modelIDs := make([]string, len(result.Data))
		for i, m := range result.Data {
			modelIDs[i] = m.ID
		}

		return modelsFetchedMsg{models: modelIDs}
	}
}

func loadInitialContextCmd(sess *session.Session) tea.Cmd {
	return func() tea.Msg {
		err := sess.LoadContext()
		return initialContextLoadedMsg{err: err}
	}
}

func scanAddFilesCmd(customExclusions []string) tea.Cmd {
	return func() tea.Msg {
		allExclusions := append([]string{}, source.Exclusions...)
		allExclusions = append(allExclusions, customExclusions...)
		items := sf.Run([]string{"."}, "", allExclusions, false)
		for i, it := range items {
			p := filepath.ToSlash(it)
			if info, err := os.Stat(it); err == nil && info.IsDir() {
				p += "/"
			}
			items[i] = p
		}
		return addFilesListResultMsg{items: items}
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
		newSess, err := session.New(sess.GetConfig(), "chat", sess.GetInstruction(), nil)
		if err != nil {
			return conversationLoadedMsg{err: err}
		}

		err = newSess.LoadConversation(filename)
		return conversationLoadedMsg{sess: newSess, err: err}
	}
}

func saveConversationCmd(sess *session.Session) tea.Cmd {
	if sess == nil {
		return nil
	}
	return func() tea.Msg {
		if err := sess.SaveConversation(); err != nil {
			log.Printf("Error saving conversation: %v", err)
			return errorMsg{sessID: sess.ID, error: err}
		}
		return nil
	}
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
		return func() tea.Msg { return errorMsg{error: err} }
	}

	if _, err := tmpfile.WriteString(content); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return func() tea.Msg { return errorMsg{error: err} }
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return func() tea.Msg { return errorMsg{error: err} }
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

func openFilesInEditorCmd(filePaths []string) tea.Cmd {
	if len(filePaths) == 0 {
		return nil
	}
	if openCmd := os.Getenv("CODER_OPEN_CMD"); openCmd != "" {
		return func() tea.Msg {
			args := append([]string{"-c", openCmd, "_"}, filePaths...)
			cmd := exec.Command("sh", args...)
			err := cmd.Run()
			return fileEditorFinishedMsg{err: err}
		}
	}
	editor := getEditor()
	cmd := exec.Command(editor, filePaths...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return fileEditorFinishedMsg{err: err}
	})
}

func getUserShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "/bin/sh"
	}
	return shell
}

func execTerminalCmd(cmdStr string) tea.Cmd {
	shell := getUserShell()
	trimmed := strings.TrimSpace(cmdStr)
	if trimmed == "" {
		c := exec.Command(shell)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return termFinishedMsg{cmdStr: "", err: err}
		})
	}

	tmpFile, err := os.CreateTemp("", "coder-term-*.log")
	if err != nil {
		c := exec.Command(shell, "-c", trimmed)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return termFinishedMsg{cmdStr: trimmed, err: err}
		})
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	wrapped := fmt.Sprintf("(%s) 2>&1 | tee %s", trimmed, tmpPath)
	c := exec.Command(shell, "-c", wrapped)
	return tea.ExecProcess(c, func(runErr error) tea.Msg {
		defer os.Remove(tmpPath)
		data, _ := os.ReadFile(tmpPath)
		output := strings.TrimSpace(string(data))
		return termFinishedMsg{cmdStr: trimmed, output: output, err: runErr}
	})
}

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

func handlePasteCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if cfg.Clipboard.PasteCmd != "" {
			data, contentType, err := clipboard.PasteCustom(cfg.Clipboard.PasteCmd)
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

		if data, _, err := clipboard.GetImageFromClipboard(); err == nil {
			if relPath, err := saveImageToRepo(data, ".png"); err == nil {
				return pasteResultMsg{isImage: true, content: relPath}
			}
		}

		content, err := clipboard.PasteText()
		if err != nil {
			return pasteResultMsg{err: fmt.Errorf("failed to read clipboard: %w", err)}
		}
		return pasteResultMsg{isImage: false, content: content}
	}
}

func saveImageToRepo(data []byte, ext string) (string, error) {
	repoRoot := project.Root()
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
