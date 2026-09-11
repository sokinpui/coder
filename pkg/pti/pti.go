package pti

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
)

const DefaultDPI = 150.0

type Options struct {
	DPI       float64
	OutputDir string
	Pages     string
	NoSave    bool
}

type PageResult struct {
	PageNumber int
	FilePath   string
	Data       []byte
}

type Result struct {
	DocName     string
	OutputDir   string
	TotalPages  int
	OutputFiles []string
	Pages       []PageResult
}

var (
	poolMu     sync.Mutex
	sharedPool pdfium.Pool
	compCache  wazero.CompilationCache
)

func maxWorkers() int {
	workers := runtime.NumCPU()
	if workers > 8 {
		return 8
	}
	if workers < 1 {
		return 1
	}
	return workers
}

func getPool() (pdfium.Pool, error) {
	poolMu.Lock()
	defer poolMu.Unlock()

	if sharedPool != nil {
		return sharedPool, nil
	}

	runtimeConfig := wazero.NewRuntimeConfig()
	if cacheDir, err := os.UserCacheDir(); err == nil {
		wazeroDir := filepath.Join(cacheDir, "coder", "wazero")
		if err := os.MkdirAll(wazeroDir, 0755); err == nil {
			if cache, err := wazero.NewCompilationCacheWithDir(wazeroDir); err == nil {
				compCache = cache
				runtimeConfig = runtimeConfig.WithCompilationCache(cache)
			}
		}
	}

	workers := maxWorkers()
	pool, err := webassembly.Init(webassembly.Config{
		RuntimeConfig: runtimeConfig,
		MinIdle:       0,
		MaxIdle:       workers,
		MaxTotal:      workers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PDFium WASM pool: %w", err)
	}

	sharedPool = pool
	return sharedPool, nil
}

func ClosePool() error {
	poolMu.Lock()
	defer poolMu.Unlock()

	var err error
	if sharedPool != nil {
		err = sharedPool.Close()
		sharedPool = nil
	}
	if compCache != nil {
		_ = compCache.Close(context.Background())
		compCache = nil
	}
	return err
}

func Convert(docPath string, opts Options) (*Result, error) {
	if opts.DPI <= 0 {
		opts.DPI = DefaultDPI
	}

	pdfBytes, err := readPDF(docPath)
	if err != nil {
		return nil, err
	}

	pool, err := getPool()
	if err != nil {
		return nil, err
	}

	instance, err := pool.GetInstance(time.Second * 30)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire PDFium instance: %w", err)
	}
	defer instance.Close()

	doc, err := instance.OpenDocument(&requests.OpenDocument{
		File: &pdfBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF %s: %w", docPath, err)
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
		Document: doc.Document,
	})

	pageCountResp, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{
		Document: doc.Document,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	totalPages := pageCountResp.PageCount
	if totalPages <= 0 {
		return nil, fmt.Errorf("PDF document %s contains no pages", docPath)
	}

	selectedPages, err := resolvePages(opts.Pages, totalPages)
	if err != nil {
		return nil, err
	}

	baseName := filepath.Base(docPath)
	docName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	var targetDir string
	if !opts.NoSave {
		targetDir = docName
		if opts.OutputDir != "" {
			targetDir = filepath.Join(opts.OutputDir, docName)
		}
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", targetDir, err)
		}
	}

	workerCount := min(len(selectedPages), maxWorkers())
	tasks := make(chan int, len(selectedPages))
	for i := range selectedPages {
		tasks <- i
	}
	close(tasks)

	results := make([]PageResult, len(selectedPages))
	var (
		errOnce sync.Once
		taskErr error
	)
	setErr := func(err error) {
		errOnce.Do(func() {
			taskErr = err
		})
	}

	runWorker := func(inst pdfium.Pdfium, openedDoc *responses.OpenDocument) {
		encoder := png.Encoder{CompressionLevel: png.BestSpeed}

		for taskIdx := range tasks {
			if taskErr != nil {
				return
			}

			pageNum := selectedPages[taskIdx]
			pageIdx := pageNum - 1

			pageRender, err := inst.RenderPageInDPI(&requests.RenderPageInDPI{
				DPI: int(opts.DPI),
				Page: requests.Page{
					ByIndex: &requests.PageByIndex{
						Document: openedDoc.Document,
						Index:    pageIdx,
					},
				},
			})
			if err != nil {
				setErr(fmt.Errorf("failed to render page %d: %w", pageNum, err))
				return
			}

			var buf bytes.Buffer
			encodeErr := encoder.Encode(&buf, pageRender.Result.Image)
			if pageRender.Cleanup != nil {
				pageRender.Cleanup()
			}
			if encodeErr != nil {
				setErr(fmt.Errorf("failed to encode PNG for page %d: %w", pageNum, encodeErr))
				return
			}

			data := buf.Bytes()
			var outFilePath string
			if targetDir != "" {
				outFilePath = filepath.Join(targetDir, fmt.Sprintf("%d.png", pageNum))
				if writeErr := os.WriteFile(outFilePath, data, 0644); writeErr != nil {
					setErr(fmt.Errorf("failed to create image file %s: %w", outFilePath, writeErr))
					return
				}
			}

			results[taskIdx] = PageResult{
				PageNumber: pageNum,
				FilePath:   outFilePath,
				Data:       data,
			}
		}
	}

	var wg sync.WaitGroup
	for w := 1; w < workerCount; w++ {
		wg.Go(func() {

			inst, err := pool.GetInstance(time.Second * 30)
			if err != nil {
				setErr(fmt.Errorf("failed to acquire PDFium instance: %w", err))
				return
			}
			defer inst.Close()

			openedDoc, err := inst.OpenDocument(&requests.OpenDocument{
				File: &pdfBytes,
			})
			if err != nil {
				setErr(fmt.Errorf("failed to open PDF document: %w", err))
				return
			}
			defer inst.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
				Document: openedDoc.Document,
			})

			runWorker(inst, openedDoc)
		})
	}

	runWorker(instance, doc)
	wg.Wait()

	if taskErr != nil {
		return nil, taskErr
	}

	var outputFiles []string
	for _, res := range results {
		if res.FilePath != "" {
			outputFiles = append(outputFiles, res.FilePath)
		}
	}

	return &Result{
		DocName:     docName,
		OutputDir:   targetDir,
		TotalPages:  totalPages,
		OutputFiles: outputFiles,
		Pages:       results,
	}, nil
}

func readPDF(filePath string) ([]byte, error) {
	if !strings.EqualFold(filepath.Ext(filePath), ".pdf") {
		return nil, fmt.Errorf("%s is not a PDF file (please convert documents to PDF first)", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, fmt.Errorf("%s is not a valid PDF document", filePath)
	}

	return data, nil
}

func resolvePages(pagesSpec string, totalPages int) ([]int, error) {
	if strings.TrimSpace(pagesSpec) == "" {
		all := make([]int, totalPages)
		for i := range totalPages {
			all[i] = i + 1
		}
		return all, nil
	}

	var pages []int
	seen := make(map[int]struct{})
	segments := strings.SplitSeq(pagesSpec, ",")

	for seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		if strings.Contains(seg, "-") {
			parts := strings.SplitN(seg, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid page range start: %s", parts[0])
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid page range end: %s", parts[1])
			}
			if start > end {
				start, end = end, start
			}
			for p := start; p <= end; p++ {
				if p < 1 || p > totalPages {
					return nil, fmt.Errorf("page %d out of bounds (1-%d)", p, totalPages)
				}
				if _, ok := seen[p]; !ok {
					seen[p] = struct{}{}
					pages = append(pages, p)
				}
			}
			continue
		}

		p, err := strconv.Atoi(seg)
		if err != nil {
			return nil, fmt.Errorf("invalid page number: %s", seg)
		}
		if p < 1 || p > totalPages {
			return nil, fmt.Errorf("page %d out of bounds (1-%d)", p, totalPages)
		}
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			pages = append(pages, p)
		}
	}

	return pages, nil
}
