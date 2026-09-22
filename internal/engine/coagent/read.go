package coagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/sokinpui/coder/internal/engine/pdf"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
)

type ReadFileTool struct{}

func (t *ReadFileTool) Declaration() ToolDeclaration {
	return ToolDeclaration{
		Type:        "function",
		Name:        "read",
		Description: "Read the content of a file from the filesystem. Supports text and code files (returns plain text), images (.png, .jpg, .jpeg, .webp, etc., loaded into visual context), and PDF documents (rendered into visual context).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The relative or absolute path of the file to read",
				},
				"pages": map[string]any{
					"type":        "string",
					"description": "Optional comma-separated page numbers or ranges to render if reading a PDF (e.g. '1-3', '1,5'). Ignored for other files.",
				},
			},
			"required": []string{"path"},
		},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, arguments string) (string, error) {
	out, _, err := t.ExecuteWithImages(ctx, arguments)
	return out, err
}

func (t *ReadFileTool) ExecuteWithImages(ctx context.Context, arguments string) (string, []types.Message, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}

	var params struct {
		Path  string `json:"path"`
		Pages string `json:"pages"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil || params.Path == "" {
		trimmed := strings.TrimSpace(arguments)
		if !strings.HasPrefix(trimmed, "{") && trimmed != "" {
			params.Path = trimmed
		} else if err != nil {
			return "", nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if strings.TrimSpace(params.Path) == "" {
		return "", nil, fmt.Errorf("path is required")
	}

	ext := strings.ToLower(filepath.Ext(params.Path))
	if ext == ".pdf" {
		return t.readPDF(params.Path, params.Pages)
	}

	if isImageExt(ext) {
		return t.readImage(params.Path)
	}

	return t.readText(params.Path)
}

func (t *ReadFileTool) readPDF(path, pages string) (string, []types.Message, error) {
	msgs, err := pdf.RenderPDFToMessages(path, pages)
	if err != nil {
		return "", nil, fmt.Errorf("failed to render PDF %s: %w", path, err)
	}

	DefaultTracker.MarkRead(path)
	return fmt.Sprintf("Successfully rendered PDF %s (%d page(s))", path, len(msgs)), msgs, nil
}

func (t *ReadFileTool) readImage(path string) (string, []types.Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}

	if len(data) == 0 {
		return "", nil, fmt.Errorf("image file %s is empty", path)
	}

	relPath := path
	if repoRoot := project.Root(); repoRoot != "" {
		if rel, err := filepath.Rel(repoRoot, path); err == nil {
			relPath = rel
		}
	}

	DefaultTracker.MarkRead(path)

	imgMsg := types.Message{
		Type:    types.ImageMessage,
		Content: filepath.ToSlash(relPath),
		Data:    data,
	}

	return fmt.Sprintf("Successfully read image %s (%d bytes)", path, len(data)), []types.Message{imgMsg}, nil
}

func (t *ReadFileTool) readText(path string) (string, []types.Message, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}

	if bytes.Contains(data, []byte{0}) || !utf8.Valid(data) {
		return "", nil, fmt.Errorf("cannot read %s: not a valid plain text file", path)
	}

	DefaultTracker.MarkRead(path)
	if len(data) == 0 {
		return "(empty file)", nil, nil
	}

	return string(data), nil, nil
}

func isImageExt(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func init() {
	DefaultRegistry.Register(&ReadFileTool{})
}
