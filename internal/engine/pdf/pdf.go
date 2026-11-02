package pdf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/pkg/pti"
)

type pdfCacheKey struct {
	path      string
	pagesSpec string
}

type pdfCacheEntry struct {
	modTime  time.Time
	size     int64
	messages []types.Message
}

type manifestEntry struct {
	ModTime  int64    `json:"mod_time"`
	Size     int64    `json:"size"`
	RelPaths []string `json:"rel_paths"`
}

var (
	cacheMu  sync.RWMutex
	memCache = make(map[pdfCacheKey]pdfCacheEntry)
)

func loadDiskManifest(manifestPath string) map[string]manifestEntry {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return make(map[string]manifestEntry)
	}
	var entries map[string]manifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return make(map[string]manifestEntry)
	}
	return entries
}

func saveDiskManifest(manifestPath string, entries map[string]manifestEntry) {
	data, err := json.Marshal(entries)
	if err != nil {
		return
	}
	_ = os.WriteFile(manifestPath, data, 0644)
}

func RenderPDFToMessages(pdfPath string, pagesSpec string) ([]types.Message, error) {
	repoRoot := project.Root()
	imagesBaseDir := filepath.Join(repoRoot, ".coder", "images")
	if err := os.MkdirAll(imagesBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create images directory: %w", err)
	}

	absPath, err := filepath.Abs(pdfPath)
	if err != nil {
		absPath = filepath.Clean(pdfPath)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		info, err = os.Stat(pdfPath)
		if err != nil {
			return nil, err
		}
		if p, err := filepath.Abs(pdfPath); err == nil {
			absPath = p
		}
	}

	key := pdfCacheKey{
		path:      filepath.ToSlash(absPath),
		pagesSpec: pagesSpec,
	}

	cacheMu.RLock()
	entry, ok := memCache[key]
	if ok && entry.modTime.Equal(info.ModTime()) && entry.size == info.Size() {
		msgs := make([]types.Message, len(entry.messages))
		copy(msgs, entry.messages)
		cacheMu.RUnlock()
		return msgs, nil
	}
	cacheMu.RUnlock()

	manifestPath := filepath.Join(imagesBaseDir, ".manifest.json")
	manifestKey := fmt.Sprintf("%s\x00%s", filepath.ToSlash(absPath), pagesSpec)

	cacheMu.Lock()
	defer cacheMu.Unlock()

	if entry, ok := memCache[key]; ok && entry.modTime.Equal(info.ModTime()) && entry.size == info.Size() {
		msgs := make([]types.Message, len(entry.messages))
		copy(msgs, entry.messages)
		return msgs, nil
	}

	diskManifest := loadDiskManifest(manifestPath)
	if diskEntry, found := diskManifest[manifestKey]; found {
		if diskEntry.ModTime == info.ModTime().UnixNano() && diskEntry.Size == info.Size() {
			diskMessages := make([]types.Message, 0, len(diskEntry.RelPaths))
			allFilesExist := true
			for _, relPath := range diskEntry.RelPaths {
				fullPath := filepath.Join(repoRoot, relPath)
				data, readErr := os.ReadFile(fullPath)
				if readErr != nil {
					allFilesExist = false
					break
				}
				diskMessages = append(diskMessages, types.Message{
					Type:    types.ImageMessage,
					Content: filepath.ToSlash(relPath),
					Data:    data,
				})
			}
			if allFilesExist && len(diskMessages) > 0 {
				memCache[key] = pdfCacheEntry{
					modTime:  info.ModTime(),
					size:     info.Size(),
					messages: diskMessages,
				}
				msgs := make([]types.Message, len(diskMessages))
				copy(msgs, diskMessages)
				return msgs, nil
			}
		}
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

	messages := make([]types.Message, 0, len(res.Pages))
	relPaths := make([]string, 0, len(res.Pages))
	for _, page := range res.Pages {
		relPath, err := filepath.Rel(repoRoot, page.FilePath)
		if err != nil {
			relPath = page.FilePath
		}
		slashedRelPath := filepath.ToSlash(relPath)
		messages = append(messages, types.Message{
			Type:    types.ImageMessage,
			Content: slashedRelPath,
			Data:    page.Data,
		})
		relPaths = append(relPaths, slashedRelPath)
	}

	memCache[key] = pdfCacheEntry{
		modTime:  info.ModTime(),
		size:     info.Size(),
		messages: messages,
	}

	diskManifest[manifestKey] = manifestEntry{
		ModTime:  info.ModTime().UnixNano(),
		Size:     info.Size(),
		RelPaths: relPaths,
	}
	saveDiskManifest(manifestPath, diskManifest)

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
