package markdown

import (
	"runtime"
	"strings"
	"sync"
)

type CachedRender struct {
	Lines   []string
	Content string
	Width   int
}

type RenderItem struct {
	Index   int
	Content string
}

type RenderResult struct {
	Index   int
	Content string
	Lines   []string
	Width   int
}

func BatchRender(items []RenderItem, width int) []RenderResult {
	if len(items) == 0 {
		return nil
	}

	workerCount := max(min(len(items), runtime.NumCPU()), 1)

	jobs := make(chan RenderItem, len(items))
	for _, it := range items {
		jobs <- it
	}
	close(jobs)

	results := make(chan RenderResult, len(items))
	var wg sync.WaitGroup

	for range workerCount {
		wg.Go(func() {
			renderer, _ := NewRenderer(width)
			for job := range jobs {
				var lines []string
				if renderer != nil {
					rendered, err := renderer.Render(job.Content)
					if err == nil {
						lines = strings.Split(rendered, "\n")
					}
				}
				if lines == nil {
					lines = strings.Split(job.Content, "\n")
				}
				results <- RenderResult{
					Index:   job.Index,
					Content: job.Content,
					Lines:   lines,
					Width:   width,
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	output := make([]RenderResult, 0, len(items))
	for res := range results {
		output = append(output, res)
	}
	return output
}
