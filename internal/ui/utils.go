package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/history"
	"github.com/sokinpui/coder/internal/engine/source"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/ui/markdown"
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
)

const statusBarMessageDuration = 1 * time.Second

func listenForEvents(sessID string, sub <-chan types.SessionEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-sub
		if !ok {
			return sessionFinishedMsg{sessID: sessID}
		}
		return sessionEventMsg{
			sessID: sessID,
			event:  ev,
			sub:    sub,
		}
	}
}

func renderAIMessageCmd(sessID string, msgIdx int, content string, width int) tea.Cmd {
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

		lines, err := markdown.RenderStreamMarkdown(content, width)
		if err != nil {
			lines = strings.Split(content, "\n")
		}
		return aiRenderedMsg{
			sessID:  sessID,
			msgIdx:  msgIdx,
			content: content,
			lines:   lines,
			width:   width,
		}
	}
}

func renderUncachedMessagesCmd(sessID string, messages []types.Message, cache map[int]markdown.CachedRender, width int, isStreaming bool) tea.Cmd {
	var items []markdown.RenderItem
	total := len(messages)

	for i, msg := range messages {
		if msg.Type != types.AIMessage || msg.Content == "" {
			continue
		}
		if isStreaming && i == total-1 {
			continue
		}
		if c, ok := cache[i]; ok && c.Content == msg.Content && c.Width == width {
			continue
		}
		items = append(items, markdown.RenderItem{Index: i, Content: msg.Content})
	}

	if len(items) == 0 {
		return nil
	}

	return func() tea.Msg {
		results := markdown.BatchRender(items, width)
		return markdownBatchRenderedMsg{
			sessID:  sessID,
			results: results,
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

func loadInitialContextCmd(sess engine.EngineSession) tea.Cmd {
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

func listHistoryCmd(histMgr *history.Manager, mode string) tea.Cmd {
	return func() tea.Msg {
		items, err := histMgr.ListConversationsByMode(mode)
		return historyListResultMsg{items: items, err: err}
	}
}

func loadConversationCmd(sess engine.EngineSession, filename string) tea.Cmd {
	return func() tea.Msg {
		newSess, err := sess.CreateNew("")
		if err != nil {
			return conversationLoadedMsg{err: err}
		}

		err = newSess.LoadConversation(filename)
		return conversationLoadedMsg{sess: newSess, err: err}
	}
}

func saveConversationCmd(sess engine.EngineSession) tea.Cmd {
	if sess == nil {
		return nil
	}
	return func() tea.Msg {
		if err := sess.SaveConversation(); err != nil {
			log.Printf("Error saving conversation: %v", err)
			return errorMsg{sessID: sess.GetID(), error: err}
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

func generateTitleCmd(sess engine.EngineSession, userPrompt string) tea.Cmd {
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

func execShellCmd(cmdStr string) tea.Cmd {
	shell := getUserShell()
	trimmed := strings.TrimSpace(cmdStr)
	if trimmed == "" {
		c := exec.Command(shell)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return shellFinishedMsg{cmdStr: "", err: err}
		})
	}

	tmpFile, err := os.CreateTemp("", "coder-shell-*.log")
	if err != nil {
		c := exec.Command(shell, "-c", trimmed)
		return tea.ExecProcess(c, func(err error) tea.Msg {
			return shellFinishedMsg{cmdStr: trimmed, err: err}
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
		return shellFinishedMsg{cmdStr: trimmed, output: output, err: runErr}
	})
}

func getVisibleLines(ta textarea.Model, width int, maxLines int) int {
	return CalculateVisibleLines(ta, width, maxLines)
}
