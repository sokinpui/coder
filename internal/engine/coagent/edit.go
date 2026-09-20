package coagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type EditFileTool struct{}

func (t *EditFileTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "edit",
		Description: "Replace an exact block of text (old_string) with new text (new_string) in a file. old_string must match exactly and uniquely.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The relative or absolute path of the file to edit",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "The exact existing text segment to replace. Must be unique in the file.",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "The new text segment to replace old_string with",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var params struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if strings.TrimSpace(params.Path) == "" {
		return "", fmt.Errorf("path is required")
	}

	if !DefaultTracker.HasRead(params.Path) {
		return "", fmt.Errorf("guard rejected: you must read %s using read before editing it", params.Path)
	}

	if params.OldString == "" {
		return "", fmt.Errorf("old_string cannot be empty")
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", params.Path, err)
	}

	content := string(data)
	count := strings.Count(content, params.OldString)
	if count == 0 {
		return "", fmt.Errorf("old_string not found in %s. Please re-read the file to ensure exact match", params.Path)
	}

	if count > 1 {
		return "", fmt.Errorf("old_string matched %d times in %s. Please include more surrounding context lines to make it unique", count, params.Path)
	}

	newContent := strings.Replace(content, params.OldString, params.NewString, 1)
	if err := os.WriteFile(params.Path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to save changes to %s: %w", params.Path, err)
	}

	return fmt.Sprintf("Successfully replaced text in %s", params.Path), nil
}

func init() {
	DefaultRegistry.Register(&EditFileTool{})
}
