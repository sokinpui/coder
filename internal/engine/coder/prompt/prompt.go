package prompt

import _ "embed"

const (
	ChatInstructions        = "You are a helpful assistant."
	ProjectSourceCodeHeader = "# PROJECT SOURCE CODE\n\n"
)

//go:embed Instructions.md
var Instructions string

//go:embed titleGenerate.md
var TitleGenerationPrompt string
