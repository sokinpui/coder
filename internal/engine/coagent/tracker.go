package coagent

import (
	"path/filepath"
	"sync"
)

type FileTracker struct {
	mu        sync.RWMutex
	readFiles map[string]struct{}
}

func NewFileTracker() *FileTracker {
	return &FileTracker{
		readFiles: make(map[string]struct{}),
	}
}

func (t *FileTracker) MarkRead(path string) {
	norm := normalizePath(path)
	if norm == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.readFiles[norm] = struct{}{}
}

func (t *FileTracker) HasRead(path string) bool {
	norm := normalizePath(path)
	if norm == "" {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.readFiles[norm]
	return ok
}

func (t *FileTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.readFiles = make(map[string]struct{})
}

func normalizePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

var DefaultTracker = NewFileTracker()
