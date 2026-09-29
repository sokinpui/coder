package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sokinpui/coder/internal/clipboard"
	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/project"
)

type PasteResultMsg struct {
	IsImage bool
	Content string
	Err     error
}

func HandlePasteCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		if cfg != nil && cfg.Clipboard.PasteCmd != "" {
			data, contentType, err := clipboard.PasteCustom(cfg.Clipboard.PasteCmd)
			if err != nil {
				return PasteResultMsg{Err: err}
			}

			if !strings.HasPrefix(contentType, "image/") {
				return PasteResultMsg{IsImage: false, Content: string(data)}
			}

			ext := ".png"
			switch contentType {
			case "image/jpeg":
				ext = ".jpg"
			case "image/webp":
				ext = ".webp"
			}

			relPath, err := SaveImageToRepo(data, ext)
			if err != nil {
				return PasteResultMsg{Err: err}
			}
			return PasteResultMsg{IsImage: true, Content: relPath}
		}

		if data, _, err := clipboard.GetImageFromClipboard(); err == nil {
			if relPath, err := SaveImageToRepo(data, ".png"); err == nil {
				return PasteResultMsg{IsImage: true, Content: relPath}
			}
		}

		content, err := clipboard.PasteText()
		if err != nil {
			return PasteResultMsg{Err: fmt.Errorf("failed to read clipboard: %w", err)}
		}
		return PasteResultMsg{IsImage: false, Content: content}
	}
}

func SaveImageToRepo(data []byte, ext string) (string, error) {
	repoRoot := project.Root()
	imagesDir := filepath.Join(repoRoot, ".coder", "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return "", fmt.Errorf("could not create images directory: %w", err)
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(imagesDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save image file: %w", err)
	}

	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return filePath, nil
	}
	return filepath.ToSlash(relPath), nil
}
