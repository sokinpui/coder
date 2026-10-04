package markdown

import (
	_ "embed"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

//go:embed my-dark.json
var myDarkTheme []byte

//go:embed my-light.json
var myLightTheme []byte

type poolKey struct {
	width  int
	isDark bool
}

var (
	poolMu sync.RWMutex
	pools  = make(map[poolKey]*sync.Pool)
)

func getRendererPool(width int) *sync.Pool {
	if width <= 0 {
		width = 80
	}

	isDark := lipgloss.HasDarkBackground()
	key := poolKey{width: width, isDark: isDark}

	poolMu.RLock()
	p, ok := pools[key]
	poolMu.RUnlock()
	if ok {
		return p
	}

	poolMu.Lock()
	defer poolMu.Unlock()
	if p, ok := pools[key]; ok {
		return p
	}

	themeBytes := myDarkTheme
	if !isDark {
		themeBytes = myLightTheme
	}

	p = &sync.Pool{
		New: func() any {
			tr, err := glamour.NewTermRenderer(
				glamour.WithStylesFromJSONBytes(themeBytes),
				glamour.WithWordWrap(width),
			)
			if err != nil {
				return nil
			}
			return tr
		},
	}
	pools[key] = p
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
	rendered, err := tr.Render(content)
	if err != nil {
		return rendered, err
	}
	return ExpandTabs(rendered, 4), nil
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

func ExpandTabs(s string, tabWidth int) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	if tabWidth <= 0 {
		tabWidth = 4
	}

	var sb strings.Builder
	sb.Grow(len(s) + 16)

	col := 0
	i := 0
	n := len(s)

	for i < n {
		b := s[i]

		if b == '\n' || b == '\r' {
			sb.WriteByte(b)
			col = 0
			i++
			continue
		}

		if b == '\x1b' {
			seqLen := ansiSequenceLength(s[i:])
			sb.WriteString(s[i : i+seqLen])
			i += seqLen
			continue
		}

		if b == '\t' {
			spaces := tabWidth - (col % tabWidth)
			for range spaces {
				sb.WriteByte(' ')
			}
			col += spaces
			i++
			continue
		}

		if b < 128 {
			sb.WriteByte(b)
			if b >= 32 {
				col++
			}
			i++
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		sb.WriteString(s[i : i+size])
		col += ansi.StringWidth(string(r))
		i += size
	}

	return sb.String()
}

func ansiSequenceLength(s string) int {
	if len(s) < 2 {
		return len(s)
	}

	switch s[1] {
	case '[':
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7E {
				return i + 1
			}
		}
		return len(s)
	case ']':
		for i := 2; i < len(s); i++ {
			if s[i] == 0x07 {
				return i + 1
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
				return i + 2
			}
		}
		return len(s)
	default:
		return 2
	}
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
