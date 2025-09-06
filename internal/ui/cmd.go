package ui

import (
	"errors"
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

// renderTick is a command that sends a renderTickMsg.
func renderTick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return renderTickMsg{}
	})
}
