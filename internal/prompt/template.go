package prompt

import _ "embed"

//go:embed Instructions.md
var CoderInstructions string

//go:embed titleGenerate.md
var TitleGenerationPrompt string

const (
	ChatInstructions          = "You are a helpful assistant."
	ProjectSourceCodeHeader   = "# PROJECT SOURCE CODE\n\n"
	ConversationHistoryHeader = "# CONVERSATION HISTORY\n\n"
)
