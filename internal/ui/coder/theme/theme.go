package theme

import (
	_ "embed"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

//go:embed dark.json
var darkJSON []byte

//go:embed light.json
var lightJSON []byte

func MarkdownStyleJSON() []byte {
	if !lipgloss.HasDarkBackground() {
		return lightJSON
	}
	return darkJSON
}

func NewRenderer(width int) (*glamour.TermRenderer, error) {
	return glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(MarkdownStyleJSON()),
		glamour.WithWordWrap(width),
	)
}
