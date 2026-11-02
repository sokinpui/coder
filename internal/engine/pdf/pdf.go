package pdf

import (
	"container/list"
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

const (
	defaultMaxCacheEntries = 16
	defaultMaxCacheBytes   = 64 * 1024 * 1024
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

type cacheItem struct {
	key      pdfCacheKey
	entry    pdfCacheEntry
	byteSize int64
}

type memoryCache struct {
	mu         sync.Mutex
	items      map[pdfCacheKey]*list.Element
	evictList  *list.List
	maxEntries int
	maxBytes   int64
	totalBytes int64
}

func newMemoryCache(maxEntries int, maxBytes int64) *memoryCache {
	return &memoryCache{
		items:      make(map[pdfCacheKey]*list.Element),
		evictList:  list.New(),
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
	}
}

func (c *memoryCache) Get(key pdfCacheKey, modTime time.Time, size int64) ([]types.Message, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, found := c.items[key]
	if !found {
		return nil, false
	}

	item := elem.Value.(*cacheItem)
	if !item.entry.modTime.Equal(modTime) || item.entry.size != size {
		c.removeElement(elem)
		return nil, false
	}

	c.evictList.MoveToFront(elem)
	msgs := make([]types.Message, len(item.entry.messages))
	copy(msgs, item.entry.messages)
	return msgs, true
}

func (c *memoryCache) Put(key pdfCacheKey, entry pdfCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entryBytes := calculateEntrySize(entry.messages)

	if elem, found := c.items[key]; found {
		oldItem := elem.Value.(*cacheItem)
		c.totalBytes -= oldItem.byteSize
		oldItem.entry = entry
		oldItem.byteSize = entryBytes
		c.totalBytes += entryBytes
		c.evictList.MoveToFront(elem)
		c.evict()
		return
	}

	item := &cacheItem{
		key:      key,
		entry:    entry,
		byteSize: entryBytes,
	}
	elem := c.evictList.PushFront(item)
	c.items[key] = elem
	c.totalBytes += entryBytes

	c.evict()
}

func (c *memoryCache) evict() {
	for (len(c.items) > c.maxEntries || (c.totalBytes > c.maxBytes && len(c.items) > 1)) && c.evictList.Len() > 0 {
		back := c.evictList.Back()
		if back == nil {
			break
		}
		c.removeElement(back)
	}
}

func (c *memoryCache) removeElement(elem *list.Element) {
	c.evictList.Remove(elem)
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
	c.totalBytes -= item.byteSize
}

func calculateEntrySize(messages []types.Message) int64 {
	var total int64
	for _, msg := range messages {
		total += int64(len(msg.Data)) + int64(len(msg.Content))
	}
	return total
}

type manifestEntry struct {
	ModTime  int64    `json:"mod_time"`
	Size     int64    `json:"size"`
	RelPaths []string `json:"rel_paths"`
}

var (
	manifestMu sync.Mutex
	memCache   = newMemoryCache(defaultMaxCacheEntries, defaultMaxCacheBytes)
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
	if msgs, ok := memCache.Get(key, info.ModTime(), info.Size()); ok {
		return msgs, nil
	}

	manifestPath := filepath.Join(imagesBaseDir, ".manifest.json")
	manifestKey := fmt.Sprintf("%s\x00%s", filepath.ToSlash(absPath), pagesSpec)

	manifestMu.Lock()
	defer manifestMu.Unlock()
	if msgs, ok := memCache.Get(key, info.ModTime(), info.Size()); ok {
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
				memCache.Put(key, pdfCacheEntry{
					modTime:  info.ModTime(),
					size:     info.Size(),
					messages: diskMessages,
				})
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

	memCache.Put(key, pdfCacheEntry{
		modTime:  info.ModTime(),
		size:     info.Size(),
		messages: messages,
	})

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
