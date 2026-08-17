package types

type EventType int

const (
	NoOp EventType = iota
	MessagesUpdated
	GenerationStarted
	AtomicMsgModeStarted
	GenerateModeStarted
	EditModeStarted
	BranchModeStarted
	HistoryModeStarted
	ActiveModeStarted
	NewSessionStarted
	FzfModeStarted
	ExternalEditorStarted
	HelpViewerStarted
	ConfigViewerStarted
	ModelViewerStarted
	ListViewerStarted
	FileViewerStarted
	Quit
)

// Event is returned by session methods to inform the UI about what happened.
type Event struct {
	Type EventType
	Data any // Can be a stream channel for GenerationStarted or an error for ErrorOccurred
	Mode string
}
