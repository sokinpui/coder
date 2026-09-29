package markdown

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

type Renderer struct {
	width int
	term  *glamour.TermRenderer
}

func NewRenderer(width int) (*Renderer, error) {
	if width <= 0 {
		width = 80
	}
	term, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(StyleJSON()),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	return &Renderer{width: width, term: term}, nil
}

func (r *Renderer) Render(content string) (string, error) {
	if r == nil || r.term == nil || content == "" {
		return content, nil
	}
	return r.term.Render(content)
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
	renderer, err := NewRenderer(width)
	if err != nil {
		return content
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimRight(rendered, "\n")
}

func RenderLines(content string, width int) ([]string, error) {
	renderer, err := NewRenderer(width)
	if err != nil {
		return strings.Split(content, "\n"), err
	}
	return renderer.RenderLines(content)
}
