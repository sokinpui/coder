package prompt

import _ "embed"

//go:embed Instructions.md
var CoderInstructions string

//go:embed titleGenerate.md
var TitleGenerationPrompt string

const (
	ProjectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	ConversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
	Separator                 = "\n\n---\n\n"
)
