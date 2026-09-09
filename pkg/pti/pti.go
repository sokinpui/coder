package pti

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
)

const DefaultDPI = 150.0

type Options struct {
	DPI       float64
	OutputDir string
	Pages     string
}

type Result struct {
	DocName     string
	OutputDir   string
	TotalPages  int
	OutputFiles []string
}

func Convert(docPath string, opts Options) (*Result, error) {
	if opts.DPI <= 0 {
		opts.DPI = DefaultDPI
	}

	pdfBytes, err := readPDF(docPath)
	if err != nil {
		return nil, err
	}

	pool, err := webassembly.Init(webassembly.Config{
		MinIdle:  1,
		MaxIdle:  1,
		MaxTotal: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PDFium WASM pool: %w", err)
	}
	defer pool.Close()

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

	targetDir := docName
	if opts.OutputDir != "" {
		targetDir = filepath.Join(opts.OutputDir, docName)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	var outputFiles []string
	for _, pageNum := range selectedPages {
		pageIdx := pageNum - 1

		pageRender, err := instance.RenderPageInDPI(&requests.RenderPageInDPI{
			DPI: int(opts.DPI),
			Page: requests.Page{
				ByIndex: &requests.PageByIndex{
					Document: doc.Document,
					Index:    pageIdx,
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to render page %d: %w", pageNum, err)
		}

		outFileName := fmt.Sprintf("%d.png", pageNum)
		outFilePath := filepath.Join(targetDir, outFileName)

		outFile, err := os.Create(outFilePath)
		if err != nil {
			if pageRender.Cleanup != nil {
				pageRender.Cleanup()
			}
			return nil, fmt.Errorf("failed to create image file %s: %w", outFilePath, err)
		}

		err = png.Encode(outFile, pageRender.Result.Image)
		outFile.Close()

		if pageRender.Cleanup != nil {
			pageRender.Cleanup()
		}

		if err != nil {
			return nil, fmt.Errorf("failed to encode PNG %s: %w", outFilePath, err)
		}

		outputFiles = append(outputFiles, outFilePath)
	}

	return &Result{
		DocName:     docName,
		OutputDir:   targetDir,
		TotalPages:  totalPages,
		OutputFiles: outputFiles,
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
