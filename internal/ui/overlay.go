package ui

// Overlay defines the interface for a UI overlay.
type Overlay interface {
	// IsVisible determines if the overlay should be displayed based on the main model's state.
	IsVisible(main *Model) bool
	// View renders the overlay on top of the main view.
	View(main *Model) string
}
