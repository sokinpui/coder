package markdown

import (
	_ "embed"

	"github.com/charmbracelet/lipgloss"
)

//go:embed dark.json
var darkJSON []byte

//go:embed light.json
var lightJSON []byte

func StyleJSON() []byte {
	if !lipgloss.HasDarkBackground() {
		return lightJSON
	}
	return darkJSON
}
