package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/types"
)

const (
	maxShellOutputBytes = 64 * 1024
	defaultShellTimeout = 60 * time.Second
)

var alwaysInteractiveCommands = map[string]struct{}{
	"vim":    {},
	"vi":     {},
	"nvim":   {},
	"nano":   {},
	"emacs":  {},
	"pico":   {},
	"top":    {},
	"htop":   {},
	"btop":   {},
	"atop":   {},
	"less":   {},
	"more":   {},
	"most":   {},
	"fzf":    {},
	"peco":   {},
	"ssh":    {},
	"telnet": {},
	"tmux":   {},
	"screen": {},
	"mosh":   {},
	"gdb":    {},
	"lldb":   {},
}

var zeroArgInteractiveCommands = map[string]struct{}{
	"python":     {},
	"python3":    {},
	"node":       {},
	"nodejs":     {},
	"irb":        {},
	"ruby":       {},
	"lua":        {},
	"php":        {},
	"sh":         {},
	"bash":       {},
	"zsh":        {},
	"fish":       {},
	"powershell": {},
	"pwsh":       {},
	"ghci":       {},
	"julia":      {},
	"R":          {},
	"sqlite3":    {},
	"mysql":      {},
	"psql":       {},
	"mongosh":    {},
	"redis-cli":  {},
}

func init() {
	registerCommand("sh", shCmd, "run non-interactive shell command", PathArgumentCompleter)
	registerCommand("term", termCmd, "run interactive terminal command or open subshell", PathArgumentCompleter)
}

func shCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /sh <command>"}, false
	}

	if blockedCmd, blocked := detectInteractiveCommand(trimmed); blocked {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: fmt.Sprintf("Error: '%s' is an interactive command. Use /term to run interactive commands.", blockedCmd),
			IsShell: true,
		}, false
	}

	output, err := RunSafeShellCommand(trimmed, defaultShellTimeout)
	success := err == nil
	if err != nil && output == "" {
		output = fmt.Sprintf("Error: %v", err)
	}

	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: output,
		IsShell: true,
	}, success
}

func termCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	return CommandOutput{
		Type:    types.TermExecutionStarted,
		Payload: trimmed,
		IsShell: true,
	}, true
}

func detectInteractiveCommand(commandStr string) (string, bool) {
	for subCmd := range extractSubCommands(commandStr) {
		fields := strings.Fields(subCmd)
		if len(fields) == 0 {
			continue
		}

		baseName := filepath.Base(fields[0])
		if _, exists := alwaysInteractiveCommands[baseName]; exists {
			return baseName, true
		}

		if len(fields) == 1 {
			if _, exists := zeroArgInteractiveCommands[baseName]; exists {
				return baseName, true
			}
		}
	}
	return "", false
}

func extractSubCommands(commandStr string) func(func(string) bool) {
	return func(yield func(string) bool) {
		delimiters := []string{"|", "&&", "||", ";"}
		parts := []string{commandStr}

		for _, delim := range delimiters {
			var next []string
			for _, part := range parts {
				for segment := range strings.SplitSeq(part, delim) {
					trimmed := strings.TrimSpace(segment)
					if trimmed != "" {
						next = append(next, trimmed)
					}
				}
			}
			parts = next
		}

		for _, p := range parts {
			if !yield(p) {
				return
			}
		}
	}
}

func RunSafeShellCommand(commandStr string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", commandStr)
	cmd.Stdin = bytes.NewReader(nil)
	cmd.Env = append(os.Environ(),
		"CI=true",
		"DEBIAN_FRONTEND=noninteractive",
		"PAGER=cat",
		"GIT_PAGER=cat",
		"TERM=dumb",
	)

	var buf bytes.Buffer
	limitWriter := &limitedWriter{w: &buf, limit: maxShellOutputBytes}
	cmd.Stdout = limitWriter
	cmd.Stderr = limitWriter

	err := cmd.Run()
	output := strings.TrimSpace(buf.String())
	if limitWriter.truncated {
		output += "\n[output truncated...]"
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %v", timeout)
	}

	return output, err
}

type limitedWriter struct {
	w         io.Writer
	limit     int
	written   int
	truncated bool
}

func (l *limitedWriter) Write(p []byte) (n int, err error) {
	if l.written >= l.limit {
		l.truncated = true
		return len(p), nil
	}
	remaining := l.limit - l.written
	if len(p) > remaining {
		n, err = l.w.Write(p[:remaining])
		l.written += n
		l.truncated = true
		return len(p), err
	}
	n, err = l.w.Write(p)
	l.written += n
	return n, err
}
