package pdf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/internal/utils"
	"github.com/sokinpui/coder/pkg/pti"
)

func RenderPDFToMessages(pdfPath string, pagesSpec string) ([]types.Message, error) {
	repoRoot := utils.GetProjectRoot()
	imagesBaseDir := filepath.Join(repoRoot, ".coder", "images")
	if err := os.MkdirAll(imagesBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create images directory: %w", err)
	}

	opts := pti.Options{
		DPI:       pti.DefaultDPI,
		OutputDir: imagesBaseDir,
		Pages:     pagesSpec,
	}

	res, err := pti.Convert(pdfPath, opts)
	if err != nil {
		return nil, err
	}

	messages := make([]types.Message, 0, len(res.OutputFiles))
	for _, outFile := range res.OutputFiles {
		data, err := os.ReadFile(outFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read rendered image %s: %w", outFile, err)
		}

		relPath, err := filepath.Rel(repoRoot, outFile)
		if err != nil {
			relPath = outFile
		}

		messages = append(messages, types.Message{
			Type:    types.ImageMessage,
			Content: filepath.ToSlash(relPath),
			Data:    data,
		})
	}

	return messages, nil
}

func RenderPDFs(pdfPaths []string) ([]types.Message, error) {
	var allMessages []types.Message
	for _, p := range pdfPaths {
		msgs, err := RenderPDFToMessages(p, "")
		if err != nil {
			return nil, err
		}
		allMessages = append(allMessages, msgs...)
	}
	return allMessages, nil
}
