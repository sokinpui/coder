package coagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

type ReadFileTool struct{}

func (t *ReadFileTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "read",
		Description: "Read the plain text content of a file from the filesystem.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The relative or absolute path of the file to read",
				},
			},
			"required": []string{"path"},
		},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var params struct {
		Path string `json:"path"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil || params.Path == "" {
		trimmed := strings.TrimSpace(arguments)
		if !strings.HasPrefix(trimmed, "{") && trimmed != "" {
			params.Path = trimmed
		} else if err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if strings.TrimSpace(params.Path) == "" {
		return "", fmt.Errorf("path is required")
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return "", err
	}

	if bytes.Contains(data, []byte{0}) || !utf8.Valid(data) {
		return "", fmt.Errorf("cannot read %s: not a valid plain text file", params.Path)
	}

	if len(data) == 0 {
		DefaultTracker.MarkRead(params.Path)
		return "(empty file)", nil
	}

	DefaultTracker.MarkRead(params.Path)
	return string(data), nil
}

func init() {
	DefaultRegistry.Register(&ReadFileTool{})
}
