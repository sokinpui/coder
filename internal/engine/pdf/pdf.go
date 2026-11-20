package pdf

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/project"
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/coder/pkg/pti"
)

const (
	ModeNative = config.PDFModeNative
	ModeImage  = config.PDFModeImage
)

type contextKey struct{}
type modeContextKey struct{}

var sessionIDKey = contextKey{}
var pdfModeKey = modeContextKey{}

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

func WithMode(ctx context.Context, mode string) context.Context {
	return context.WithValue(ctx, pdfModeKey, config.NormalizePDFMode(mode))
}

func ModeFromContext(ctx context.Context) string {
	if ctx == nil {
		return ModeNative
	}
	val, ok := ctx.Value(pdfModeKey).(string)
	if !ok {
		return ModeNative
	}
	return config.NormalizePDFMode(val)
}

func NormalizeMode(mode string) string {
	return config.NormalizePDFMode(mode)
}

var (
	pageCacheMu sync.RWMutex
	pageCache   = make(map[string][]types.PDFPage)
)

func init() {
	types.RegisterPDFResolver(ResolvePDF)
}

func EvictCache(cacheDir string) {
	pageCacheMu.Lock()
	delete(pageCache, cacheDir)
	pageCacheMu.Unlock()
}

func ResolvePDF(msg types.Message, mode string) types.PDFResolved {
	meta, ok := types.ParsePDFMetadata(msg.Content)
	if !ok {
		return types.PDFResolved{}
	}

	effectiveMode := NormalizeMode(mode)
	if strings.TrimSpace(mode) == "" {
		effectiveMode = NormalizeMode(meta.Mode)
	}

	if effectiveMode == ModeImage {
		if len(meta.Files) > 0 {
			if pages := GetPages(meta); len(pages) > 0 {
				return types.PDFResolved{Filename: meta.Filename, Pages: pages}
			}
		}
		if resolvedPath := resolveReadablePath(meta.Path); resolvedPath != "" {
			if renderedMsg, err := RenderPDF("", resolvedPath, meta.Pages); err == nil {
				if renderedMeta, ok := types.ParsePDFMetadata(renderedMsg.Content); ok {
					return types.PDFResolved{
						Filename: renderedMeta.Filename,
						Pages:    GetPages(renderedMeta),
					}
				}
			}
		}
		return types.PDFResolved{Filename: meta.Filename}
	}

	if len(msg.Data) > 0 {
		return types.PDFResolved{Filename: meta.Filename, RawData: msg.Data}
	}
	if data := LoadRawPDFBytes(meta.Path); len(data) > 0 {
		return types.PDFResolved{Filename: meta.Filename, RawData: data}
	}
	if len(meta.Files) > 0 {
		if pages := GetPages(meta); len(pages) > 0 {
			return types.PDFResolved{Filename: meta.Filename, Pages: pages}
		}
	}
	return types.PDFResolved{Filename: meta.Filename}
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

func LoadNativePDF(pdfPath string) (types.Message, error) {
	if !strings.EqualFold(filepath.Ext(pdfPath), ".pdf") {
		return types.Message{}, fmt.Errorf("%s is not a PDF file (please convert documents to PDF first)", pdfPath)
	}

	absPath, info, err := resolvePDFInfo(pdfPath)
	if err != nil {
		return types.Message{}, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		data, err = os.ReadFile(pdfPath)
		if err != nil {
			return types.Message{}, fmt.Errorf("failed to read %s: %w", pdfPath, err)
		}
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return types.Message{}, fmt.Errorf("%s is not a valid PDF document", pdfPath)
	}

	meta := types.PDFMetadata{
		Filename: filepath.Base(pdfPath),
		Path:     storePDFPath(absPath, pdfPath),
		Mode:     ModeNative,
		Size:     info.Size(),
	}

	msg, err := createPDFMessage(meta)
	if err != nil {
		return types.Message{}, err
	}
	msg.Data = data
	return msg, nil
}

func LoadRawPDFBytes(path string) []byte {
	resolved := resolveReadablePath(path)
	if resolved == "" {
		return nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil
	}
	return data
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

	meta, pdfPages := buildRenderMetadata(repoRoot, absPath, pdfPath, pagesSpec, cacheRelDir, info.Size(), res.Pages)
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

func resolveReadablePath(path string) string {
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if !filepath.IsAbs(path) {
		if root := project.Root(); root != "" {
			candidate := filepath.Join(root, path)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return ""
}

func storePDFPath(absPath, fallback string) string {
	if root := project.Root(); root != "" {
		if rel, err := filepath.Rel(root, absPath); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(fallback)
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

func buildRenderMetadata(repoRoot, absPath, originalPath, pagesSpec, cacheRelDir string, size int64, pages []pti.PageResult) (types.PDFMetadata, []types.PDFPage) {
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
		Path:      storePDFPath(absPath, originalPath),
		Pages:     pagesSpec,
		CacheDir:  cacheRelDir,
		PageCount: len(pages),
		Files:     relFiles,
		Mode:      ModeImage,
		Size:      size,
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
