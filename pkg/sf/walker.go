package sf

import (
	"os"
	"path/filepath"
	"sync"
)

type Engine struct {
	concurrency chan struct{}
	wg          sync.WaitGroup
	fileType    string
	excludes    []string
	showHidden  bool
}

func NewEngine(maxConcurrency int, fileType string, excludes []string, showHidden bool) *Engine {
	return &Engine{
		concurrency: make(chan struct{}, maxConcurrency),
		fileType:    fileType,
		excludes:    excludes,
		showHidden:  showHidden,
	}
}

func (e *Engine) Walk(roots []string, results chan<- string) {
	for _, root := range roots {
		matcher := NewMatcher(root, e.excludes, e.showHidden)
		e.wg.Add(1)

		go func(r string, m *Matcher) {
			defer e.wg.Done()

			info, err := os.Lstat(r)
			if err != nil {
				return
			}

			if info.IsDir() {
				e.spawnWorker(r, results, m)
			} else if e.isTypeMatch(info.IsDir()) {
				results <- r
			}
		}(root, matcher)
	}

	go func() {
		e.wg.Wait()
		close(results)
	}()
}

func (e *Engine) walkDir(path string, results chan<- string, matcher *Matcher) {
	defer e.wg.Done()

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if matcher.ShouldSkip(fullPath, entry) {
			continue
		}

		if e.isTypeMatch(entry.IsDir()) {
			results <- fullPath
		}

		if entry.IsDir() {
			e.spawnWorker(fullPath, results, matcher)
		}
	}
}

func (e *Engine) spawnWorker(path string, results chan<- string, matcher *Matcher) {
	e.wg.Add(1)
	select {
	case e.concurrency <- struct{}{}:
		go func() {
			e.walkDir(path, results, matcher)
			<-e.concurrency
		}()
	default:
		e.walkDir(path, results, matcher)
	}
}

func (e *Engine) isTypeMatch(isDir bool) bool {
	if e.fileType == "" {
		return true
	}

	switch e.fileType {
	case "file":
		return !isDir
	case "dir":
		return isDir
	}
	return false
}
