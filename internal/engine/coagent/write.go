package coagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WriteFileTool struct{}

func (t *WriteFileTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "write",
		Description: "Write content to a file. Creates the file if it does not exist, or completely overwrites it if it does.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The relative or absolute path of the file to write",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The full content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if strings.TrimSpace(params.Path) == "" {
		return "", fmt.Errorf("path is required")
	}

	dir := filepath.Dir(params.Path)
	if dir != "." && dir != "/" && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", params.Path, err)
	}

	DefaultTracker.MarkRead(params.Path)
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(params.Content), params.Path), nil
}

func init() {
	DefaultRegistry.Register(&WriteFileTool{})
}
