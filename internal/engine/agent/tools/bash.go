package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultBashTimeout = 60 * time.Second
	maxOutputBytes     = 64 * 1024
)

type BashTool struct{}

func (t *BashTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "bash",
		Description: "Execute a bash shell command on the host system. Returns combined stdout and stderr.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The bash shell command to execute",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (t *BashTool) Execute(ctx context.Context, arguments string) (string, error) {
	return t.ExecuteStream(ctx, arguments, nil)
}

func (t *BashTool) ExecuteStream(ctx context.Context, arguments string, outChan chan<- string) (string, error) {
	var params struct {
		Command string `json:"command"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil || params.Command == "" {
		trimmedArgs := strings.TrimSpace(arguments)
		if !strings.HasPrefix(trimmedArgs, "{") && trimmedArgs != "" {
			params.Command = trimmedArgs
		} else if err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if strings.TrimSpace(params.Command) == "" {
		return "", fmt.Errorf("command is required")
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		runCtx, cancel = context.WithTimeout(ctx, defaultBashTimeout)
		defer cancel()
	}

	shell := "bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}

	pr, pw := io.Pipe()
	cmd := exec.CommandContext(runCtx, shell, "-c", params.Command)
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	var fullBuf bytes.Buffer
	var fullMu sync.Mutex
	readDone := make(chan struct{})

	go func() {
		defer close(readDone)
		buf := make([]byte, 4096)
		for {
			n, rErr := pr.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				fullMu.Lock()
				if fullBuf.Len() < maxOutputBytes {
					fullBuf.Write(buf[:n])
				}
				fullMu.Unlock()
				if outChan != nil {
					select {
					case outChan <- chunk:
					case <-runCtx.Done():
						return
					}
				}
			}
			if rErr != nil {
				break
			}
		}
	}()

	err := cmd.Wait()
	pw.Close()
	<-readDone
	pr.Close()

	outputBytes := fullBuf.Bytes()
	output := string(outputBytes)
	if len(output) > maxOutputBytes {
		output = output[:maxOutputBytes] + "\n...[output truncated]"
	}

	trimmedOutput := strings.TrimSpace(output)
	if err != nil {
		if trimmedOutput != "" {
			return fmt.Sprintf("%s\nCommand exited with error: %v", trimmedOutput, err), nil
		}
		return fmt.Sprintf("Command exited with error: %v", err), nil
	}

	if trimmedOutput == "" {
		return "(no output)", nil
	}
	return trimmedOutput, nil
}

func init() {
	DefaultRegistry.Register(&BashTool{})
}
