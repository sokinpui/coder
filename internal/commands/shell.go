package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

const (
	maxShellOutputBytes = 64 * 1024
	defaultShellTimeout = 60 * time.Second
)

func init() {
	registerCommand("sh", shCmd, "run non-interactive shell command", PathArgumentCompleter)
	registerCommand("term", termCmd, "run interactive terminal command or open subshell", PathArgumentCompleter)
}

func shCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "Usage: /sh <command>"}, false
	}

	output, err := RunSafeShellCommand(trimmed, defaultShellTimeout)
	success := err == nil
	if err != nil && output == "" {
		output = fmt.Sprintf("Error: %v", err)
	}

	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: output,
		Metadata: map[string]any{
			"canAISee": true,
			"isShell":  true,
		},
	}, success
}

func termCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	return CommandOutput{
		Type:    types.TermExecutionStarted,
		Payload: trimmed,
		Metadata: map[string]any{
			"canAISee": true,
			"isShell":  true,
		},
	}, true
}

func RegisterShellCommands(cfg *config.Config) []string {
	var errors []string
	if cfg.ShellCommands == nil {
		return nil
	}

	for name, cmdDef := range cfg.ShellCommands {
		if IsBuiltIn(name) {
			errors = append(errors, fmt.Sprintf("Shell command '%s' conflicts with a built-in command.", name))
			continue
		}

		definition := cmdDef // capture for closure
		commandName := name

		registerCommand(commandName, func(args string, s SessionController) (CommandOutput, bool) {
			execStr := definition.Exec
			argList := strings.Fields(args)

			for i, arg := range argList {
				placeholder := fmt.Sprintf("$%d", i+1)
				execStr = strings.ReplaceAll(execStr, placeholder, arg)
			}

			for i := len(argList); i < 10; i++ {
				placeholder := fmt.Sprintf("$%d", i+1)
				if strings.Contains(execStr, placeholder) {
					execStr = strings.ReplaceAll(execStr, placeholder, "")
				}
			}

			out, err := RunSafeShellCommand(execStr, defaultShellTimeout)
			payload := out
			success := err == nil
			if err != nil {
				if payload == "" {
					payload = err.Error()
				}
				success = false
			}

			return CommandOutput{
				Type:    types.MessagesUpdated,
				Payload: payload,
				Metadata: map[string]any{
					"canAISee": definition.CanAISee,
					"isShell":  true,
				},
			}, success
		}, definition.Description, nil)
	}

	return errors
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
