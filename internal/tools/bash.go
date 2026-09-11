package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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

	cmd := exec.CommandContext(runCtx, shell, "-c", params.Command)
	outputBytes, err := cmd.CombinedOutput()

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
