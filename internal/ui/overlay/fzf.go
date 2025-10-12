package overlay

import (
	"coder/internal/ui/update"

	"github.com/rmhubbert/bubbletea-overlay"
)

type FzfOverlay struct{}

func (f *FzfOverlay) IsVisible(main *update.Model) bool {
	return main.State == update.StateFzf
}

func (f *FzfOverlay) View(main *update.Model) string {
	fzfView := main.FuzzyFinder.View()
	if fzfView == "" {
		return main.View()
	}

	// Place it above the text area, similar to the command palette.

	overlayModel := overlay.New(
		main.FuzzyFinder,
		main,
		overlay.Center,
		overlay.Center,
		0,
		0,
	)
	return overlayModel.View()
}
