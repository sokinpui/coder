package markdown

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

var (
	poolMu sync.RWMutex
	pools  = make(map[int]*sync.Pool)
)

func getRendererPool(width int) *sync.Pool {
	if width <= 0 {
		width = 80
	}

	poolMu.RLock()
	p, ok := pools[width]
	poolMu.RUnlock()
	if ok {
		return p
	}

	poolMu.Lock()
	defer poolMu.Unlock()
	if p, ok := pools[width]; ok {
		return p
	}

	p = &sync.Pool{
		New: func() any {
			tr, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle("dark"),
				glamour.WithWordWrap(width),
			)
			if err != nil {
				return nil
			}
			return tr
		},
	}
	pools[width] = p
	return p
}

type Renderer struct {
	width int
}

func NewRenderer(width int) (*Renderer, error) {
	if width <= 0 {
		width = 80
	}
	p := getRendererPool(width)
	tr := p.Get()
	if tr == nil {
		return nil, fmt.Errorf("failed to create glamour renderer")
	}
	p.Put(tr)
	return &Renderer{width: width}, nil
}

func (r *Renderer) Render(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	width := 80
	if r != nil && r.width > 0 {
		width = r.width
	}
	p := getRendererPool(width)
	v := p.Get()
	if v == nil {
		return content, fmt.Errorf("failed to acquire renderer from pool")
	}
	tr := v.(*glamour.TermRenderer)
	defer p.Put(tr)
	return tr.Render(content)
}

func (r *Renderer) RenderLines(content string) ([]string, error) {
	rendered, err := r.Render(content)
	if err != nil {
		return nil, err
	}
	return strings.Split(rendered, "\n"), nil
}

func (r *Renderer) Width() int {
	if r == nil {
		return 0
	}
	return r.width
}

func Render(content string, width int) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	r, err := NewRenderer(width)
	if err != nil {
		return content
	}
	rendered, err := r.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimRight(rendered, "\n")
}

func RenderLines(content string, width int) ([]string, error) {
	r, err := NewRenderer(width)
	if err != nil {
		return strings.Split(content, "\n"), err
	}
	return r.RenderLines(content)
}

func RenderStreamMarkdown(content string, width int) ([]string, error) {
	if content == "" {
		return nil, nil
	}
	r, err := NewRenderer(width)
	if err != nil {
		return strings.Split(content, "\n"), err
	}
	return r.RenderLines(content)
}
