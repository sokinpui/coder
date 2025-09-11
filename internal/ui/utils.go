package ui

import (
	"coder/internal/contextdir"
	"coder/internal/source"
	"coder/internal/token"
	"errors"
	"fmt"
	"sync"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func loadInitialContextCmd() tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup
		var sysInstructions, docs, projSource string
		var ctxErr, srcErr error

		wg.Add(2)

		go func() {
			defer wg.Done()
			sysInstructions, docs, ctxErr = contextdir.LoadContext()
		}()

		go func() {
			defer wg.Done()
			projSource, srcErr = source.LoadProjectSource()
		}()

		wg.Wait()

		if ctxErr != nil {
			return initialContextLoadedMsg{err: fmt.Errorf("failed to load context: %w", ctxErr)}
		}
		if srcErr != nil {
			return initialContextLoadedMsg{err: fmt.Errorf("failed to load project source: %w", srcErr)}
		}

		return initialContextLoadedMsg{
			systemInstructions: sysInstructions,
			providedDocuments:  docs,
			projectSourceCode:  projSource,
		}
	}
}

func renderTick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return renderTickMsg{}
	})
}

func ctrlCTimeout() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return ctrlCTimeoutMsg{}
	})
}
