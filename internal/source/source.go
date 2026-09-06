package source

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sokinpui/coder/pkg/pcat"
)

type fileCacheEntry struct {
	modTime   time.Time
	size      int64
	formatted string
}

type SourceCache struct {
	mu      sync.RWMutex
	entries map[string]fileCacheEntry
}

var defaultCache = &SourceCache{
	entries: make(map[string]fileCacheEntry),
}

func LoadProjectSource(files []string) (string, error) {
	return defaultCache.Load(files)
}

func (c *SourceCache) Load(files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	c.mu.RLock()
	toReadIndices := make([]int, 0)
	results := make([]string, len(files))
	stats := make([]os.FileInfo, len(files))

	for i, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		stats[i] = info

		entry, found := c.entries[file]
		if found && entry.size == info.Size() && entry.modTime.Equal(info.ModTime()) {
			results[i] = entry.formatted
			continue
		}
		toReadIndices = append(toReadIndices, i)
	}
	c.mu.RUnlock()

	if len(toReadIndices) == 0 {
		return joinFormattedResults(results), nil
	}

	workerCount := max(min(len(toReadIndices), runtime.NumCPU()), 1)

	jobs := make(chan int, len(toReadIndices))
	for _, idx := range toReadIndices {
		jobs <- idx
	}
	close(jobs)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Go(func() {
			for idx := range jobs {
				file := files[idx]
				formatted := pcat.FormatSingleFile(file, false)
				results[idx] = formatted
			}
		})
	}
	wg.Wait()

	c.mu.Lock()
	for _, idx := range toReadIndices {
		if stats[idx] == nil {
			delete(c.entries, files[idx])
			continue
		}
		c.entries[files[idx]] = fileCacheEntry{
			modTime:   stats[idx].ModTime(),
			size:      stats[idx].Size(),
			formatted: results[idx],
		}
	}
	c.mu.Unlock()

	return joinFormattedResults(results), nil
}

func joinFormattedResults(results []string) string {
	var out strings.Builder
	for _, text := range results {
		out.WriteString(text)
	}
	res := strings.TrimSuffix(out.String(), "\n")
	if res == "" {
		return ""
	}
	return res + "\n---\n"
}
