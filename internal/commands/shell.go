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

var allowedCommands = map[string]struct{}{
	"ack":            {},
	"ag":             {},
	"awk":            {},
	"biome":          {},
	"black":          {},
	"bundle":         {},
	"bundler":        {},
	"bun":            {},
	"bunx":           {},
	"bzip2":          {},
	"cargo":          {},
	"cat":            {},
	"clang":          {},
	"clang++":        {},
	"clippy":         {},
	"cmake":          {},
	"command":        {},
	"composer":       {},
	"curl":           {},
	"cut":            {},
	"date":           {},
	"deno":           {},
	"df":             {},
	"diff":           {},
	"dir":            {},
	"docker":         {},
	"docker-compose": {},
	"dotnet":         {},
	"du":             {},
	"dune":           {},
	"echo":           {},
	"egrep":          {},
	"elixir":         {},
	"env":            {},
	"eslint":         {},
	"fd":             {},
	"fgrep":          {},
	"file":           {},
	"find":           {},
	"flake8":         {},
	"g++":            {},
	"gcc":            {},
	"gem":            {},
	"gh":             {},
	"git":            {},
	"glab":           {},
	"go":             {},
	"gofmt":          {},
	"golangci-lint":  {},
	"gradle":         {},
	"gradlew":        {},
	"grep":           {},
	"gunzip":         {},
	"gzip":           {},
	"head":           {},
	"helm":           {},
	"http":           {},
	"id":             {},
	"itf":            {},
	"java":           {},
	"javac":          {},
	"jq":             {},
	"just":           {},
	"kubectl":        {},
	"ls":             {},
	"make":           {},
	"mix":            {},
	"mvn":            {},
	"mypy":           {},
	"ninja":          {},
	"node":           {},
	"nodejs":         {},
	"npm":            {},
	"npx":            {},
	"nuget":          {},
	"opam":           {},
	"pcat":           {},
	"pdm":            {},
	"php":            {},
	"phpunit":        {},
	"pip":            {},
	"pip3":           {},
	"pipenv":         {},
	"pnpm":           {},
	"podman":         {},
	"poetry":         {},
	"prettier":       {},
	"printf":         {},
	"printenv":       {},
	"pwd":            {},
	"pytest":         {},
	"python":         {},
	"python3":        {},
	"rake":           {},
	"rebar3":         {},
	"rg":             {},
	"rspec":          {},
	"rubocop":        {},
	"ruby":           {},
	"ruff":           {},
	"rustc":          {},
	"sd":             {},
	"sed":            {},
	"sf":             {},
	"sort":           {},
	"stat":           {},
	"swift":          {},
	"tail":           {},
	"tar":            {},
	"task":           {},
	"tee":            {},
	"tr":             {},
	"tree":           {},
	"tsc":            {},
	"type":           {},
	"uname":          {},
	"uniq":           {},
	"unzip":          {},
	"uv":             {},
	"wc":             {},
	"wget":           {},
	"whereis":        {},
	"which":          {},
	"whoami":         {},
	"xargs":          {},
	"xz":             {},
	"yarn":           {},
	"yq":             {},
	"zig":            {},
	"zip":            {},
	"zstd":           {},
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

	if disallowedCmd, allowed := validateAllowedCommands(trimmed); !allowed {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: fmt.Sprintf("Error: '%s' is not in the allowed commands whitelist. Use /term to run arbitrary or interactive commands.", disallowedCmd),
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

func validateAllowedCommands(commandStr string) (string, bool) {
	for subCmd := range extractSubCommands(commandStr) {
		cmdName := findCommandName(subCmd)
		if cmdName == "" {
			continue
		}
		if _, ok := allowedCommands[cmdName]; !ok {
			return cmdName, false
		}
	}
	return "", true
}

func findCommandName(subCmd string) string {
	for _, field := range strings.Fields(subCmd) {
		if strings.Contains(field, "=") && !strings.HasPrefix(field, "./") && !strings.HasPrefix(field, "/") {
			continue
		}
		return filepath.Base(field)
	}
	return ""
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
