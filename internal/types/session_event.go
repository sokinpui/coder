package types

// EventType defines the type of event returned by the session.
type EventType int

const (
	// NoOp indicates that no significant action was taken.
	NoOp EventType = iota
	// MessagesUpdated indicates that the message list was updated.
	MessagesUpdated
	// GenerationStarted indicates a new AI generation task has begun.
	GenerationStarted
	// VisualModeStarted indicates the UI should enter visual mode.
	VisualModeStarted
	// GenerateModeStarted indicates the UI should enter visual generate mode.
	GenerateModeStarted
	// EditModeStarted indicates the UI should enter visual edit mode.
	EditModeStarted
	// BranchModeStarted indicates the UI should enter visual branch mode.
	BranchModeStarted
	// SearchModeStarted indicates the UI should enter search mode.
	SearchModeStarted
	// HistoryModeStarted indicates the UI should enter history browsing mode.
	HistoryModeStarted
	// NewSessionStarted indicates the session has been reset.
	NewSessionStarted
	// tree mode
	TreeModeStarted
	// fzf mode
	FzfModeStarted
	// Quit indicates the application should quit.
	Quit
)

// Event is returned by session methods to inform the UI about what happened.
type Event struct {
	Type EventType
	Data any // Can be a stream channel for GenerationStarted or an error for ErrorOccurred
}
