package pdf

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/pkg/pti"
)

type contextKey struct{}

var sessionIDKey = contextKey{}

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

func SessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	val, ok := ctx.Value(sessionIDKey).(string)
	if !ok {
		return ""
	}
	return val
}

var (
	pageCacheMu sync.RWMutex
	pageCache   = make(map[string][]types.PDFPage)
)

func init() {
	types.RegisterPDFPageResolver(ResolvePages)
}

func EvictCache(cacheDir string) {
	pageCacheMu.Lock()
	delete(pageCache, cacheDir)
	pageCacheMu.Unlock()
}

func ResolvePages(content string) (string, []types.PDFPage) {
	meta, ok := types.ParsePDFMetadata(content)
	if !ok {
		return "", nil
	}
	pages := GetPages(meta)
	return meta.Filename, pages
}

func GetPages(meta types.PDFMetadata) []types.PDFPage {
	if meta.CacheDir == "" {
		return nil
	}

	pageCacheMu.RLock()
	if cached, found := pageCache[meta.CacheDir]; found && len(cached) > 0 {
		pageCacheMu.RUnlock()
		return cached
	}
	pageCacheMu.RUnlock()

	pageCacheMu.Lock()
	defer pageCacheMu.Unlock()
	if cached, found := pageCache[meta.CacheDir]; found && len(cached) > 0 {
		return cached
	}

	repoRoot := project.Root()
	pages := make([]types.PDFPage, 0, len(meta.Files))
	for i, f := range meta.Files {
		fullPath := f
		if !filepath.IsAbs(fullPath) {
			fullPath = filepath.Join(repoRoot, fullPath)
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		pageNum := i + 1
		base := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		base = strings.TrimPrefix(base, "page_")
		base = strings.TrimPrefix(base, "page-")
		base = strings.TrimPrefix(base, "page")
		if n, err := strconv.Atoi(base); err == nil && n > 0 {
			pageNum = n
		}
		pages = append(pages, types.PDFPage{
			Index: pageNum,
			Data:  data,
		})
	}

	if len(pages) > 0 {
		pageCache[meta.CacheDir] = pages
	}
	return pages
}

func RenderPDF(sessionID, pdfPath, pagesSpec string) (types.Message, error) {
	repoRoot := project.Root()
	absPath, info, err := resolvePDFInfo(pdfPath)
	if err != nil {
		return types.Message{}, err
	}

	pdfHash := computePDFHash(absPath, pagesSpec, info.ModTime().UnixNano(), info.Size())
	sessPart := sanitizeIdentifier(sessionID)
	if sessPart == "" {
		sessPart = "default"
	}

	folderName := fmt.Sprintf("%s-%s", sessPart, pdfHash)
	cacheRelDir := filepath.ToSlash(filepath.Join(".coder", "pdf", folderName))
	cacheAbsDir := filepath.Join(repoRoot, ".coder", "pdf", folderName)

	if meta, ok := loadExistingRender(cacheAbsDir, cacheRelDir); ok {
		return createPDFMessage(meta)
	}

	if err := os.MkdirAll(cacheAbsDir, 0755); err != nil {
		return types.Message{}, fmt.Errorf("failed to create pdf cache directory: %w", err)
	}

	opts := pti.Options{
		DPI:       pti.DefaultDPI,
		OutputDir: cacheAbsDir,
		Pages:     pagesSpec,
	}

	res, err := pti.Convert(pdfPath, opts)
	if err != nil {
		return types.Message{}, err
	}
	if len(res.Pages) == 0 {
		return types.Message{}, fmt.Errorf("no pages rendered for PDF %s", pdfPath)
	}

	meta, pdfPages := buildRenderMetadata(repoRoot, pdfPath, pagesSpec, cacheRelDir, res.Pages)
	pageCacheMu.Lock()
	pageCache[cacheRelDir] = pdfPages
	pageCacheMu.Unlock()

	saveMetadata(cacheAbsDir, meta)
	return createPDFMessage(meta)
}

func RenderPDFToMessages(pdfPath string, pagesSpec string) ([]types.Message, error) {
	msg, err := RenderPDF("", pdfPath, pagesSpec)
	if err != nil {
		return nil, err
	}
	return []types.Message{msg}, nil
}

func resolvePDFInfo(path string) (string, os.FileInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = filepath.Clean(path)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		info, err = os.Stat(path)
		if err != nil {
			return "", nil, err
		}
	}
	return absPath, info, nil
}

func computePDFHash(absPath, pagesSpec string, modTimeNano, size int64) string {
	h := sha256.New()
	h.Write([]byte(absPath))
	h.Write([]byte(pagesSpec))
	h.Write([]byte(strconv.FormatInt(modTimeNano, 10)))
	h.Write([]byte(strconv.FormatInt(size, 10)))
	return fmt.Sprintf("%x", h.Sum(nil))[:12]
}

func sanitizeIdentifier(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func loadExistingRender(absDir, relDir string) (types.PDFMetadata, bool) {
	metaPath := filepath.Join(absDir, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return types.PDFMetadata{}, false
	}

	meta, ok := types.ParsePDFMetadata(string(data))
	if !ok || len(meta.Files) == 0 {
		return types.PDFMetadata{}, false
	}

	repoRoot := project.Root()
	for _, relFile := range meta.Files {
		fullFile := filepath.Join(repoRoot, relFile)
		if _, err := os.Stat(fullFile); err != nil {
			return types.PDFMetadata{}, false
		}
	}
	return meta, true
}

func buildRenderMetadata(repoRoot, originalPath, pagesSpec, cacheRelDir string, pages []pti.PageResult) (types.PDFMetadata, []types.PDFPage) {
	relFiles := make([]string, len(pages))
	pdfPages := make([]types.PDFPage, len(pages))

	for i, page := range pages {
		rel, err := filepath.Rel(repoRoot, page.FilePath)
		if err != nil {
			rel = page.FilePath
		}
		relFiles[i] = filepath.ToSlash(rel)
		pageNum := i + 1
		base := strings.TrimSuffix(filepath.Base(page.FilePath), filepath.Ext(page.FilePath))
		base = strings.TrimPrefix(base, "page_")
		base = strings.TrimPrefix(base, "page-")
		base = strings.TrimPrefix(base, "page")
		if n, err := strconv.Atoi(base); err == nil && n > 0 {
			pageNum = n
		}
		pdfPages[i] = types.PDFPage{
			Index: pageNum,
			Data:  page.Data,
		}
	}

	meta := types.PDFMetadata{
		Filename:  filepath.Base(originalPath),
		Path:      filepath.ToSlash(originalPath),
		Pages:     pagesSpec,
		CacheDir:  cacheRelDir,
		PageCount: len(pages),
		Files:     relFiles,
	}
	return meta, pdfPages
}

func saveMetadata(cacheAbsDir string, meta types.PDFMetadata) {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(cacheAbsDir, "metadata.json"), metaBytes, 0644)
}

func createPDFMessage(meta types.PDFMetadata) (types.Message, error) {
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return types.Message{}, err
	}
	return types.Message{
		Type:    types.PDFMessage,
		Content: string(metaBytes),
	}, nil
}
