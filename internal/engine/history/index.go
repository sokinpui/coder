package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const indexFileName = "index.json"

type IndexEntry struct {
	Filename   string    `json:"filename"`
	Mode       string    `json:"mode,omitempty"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type scanResult struct {
	filename string
	entry    IndexEntry
}

func (m *Manager) scanEntriesParallel(entries []os.DirEntry) map[string]IndexEntry {
	workerCount := max(min(len(entries), runtime.NumCPU()*2), 1)

	jobs := make(chan os.DirEntry, len(entries))
	for _, entry := range entries {
		jobs <- entry
	}
	close(jobs)

	resultsChan := make(chan scanResult, len(entries))
	var wg sync.WaitGroup

	for range workerCount {
		wg.Go(func() {
			for entry := range jobs {
				item, ok := m.parseEntry(entry)
				if !ok {
					continue
				}
				resultsChan <- scanResult{filename: entry.Name(), entry: item}
			}
		})
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	results := make(map[string]IndexEntry, len(entries))
	for res := range resultsChan {
		results[res.filename] = res.entry
	}
	return results
}

func (m *Manager) parseEntry(entry os.DirEntry) (IndexEntry, bool) {
	filePath := filepath.Join(m.historyPath, entry.Name())
	metadata, err := ParseFileMetadata(filePath)
	if err != nil {
		return IndexEntry{}, false
	}

	fileInfo, infoErr := entry.Info()
	if metadata.CreatedAt.IsZero() && infoErr == nil {
		metadata.CreatedAt = fileInfo.ModTime()
	}
	if metadata.ModifiedAt.IsZero() {
		if !metadata.CreatedAt.IsZero() {
			metadata.ModifiedAt = metadata.CreatedAt
		} else if infoErr == nil {
			metadata.ModifiedAt = fileInfo.ModTime()
		}
	}

	return IndexEntry{
		Filename:   entry.Name(),
		Mode:       metadata.Mode,
		Title:      metadata.Title,
		CreatedAt:  metadata.CreatedAt,
		ModifiedAt: metadata.ModifiedAt,
	}, true
}

func (m *Manager) loadIndex() map[string]IndexEntry {
	indexPath := filepath.Join(m.historyPath, indexFileName)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return make(map[string]IndexEntry)
	}

	var entries map[string]IndexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return make(map[string]IndexEntry)
	}
	return entries
}

func (m *Manager) saveIndex(entries map[string]IndexEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	indexPath := filepath.Join(m.historyPath, indexFileName)
	return os.WriteFile(indexPath, data, 0644)
}
