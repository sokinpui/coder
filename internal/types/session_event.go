package types

type EventType int

const (
	NoOp EventType = iota
	MessagesUpdated
	GenerationStarted
	VisualModeStarted
	GenerateModeStarted
	EditModeStarted
	BranchModeStarted
	SearchModeStarted
	HistoryModeStarted
	ActiveModeStarted
	NewSessionStarted
	TreeModeStarted
	FzfModeStarted
	JumpModeStarted
	Quit
)

// Event is returned by session methods to inform the UI about what happened.
type Event struct {
	Type EventType
	Data any // Can be a stream channel for GenerationStarted or an error for ErrorOccurred
	Mode string
}
