package prompt

import _ "embed"

//go:embed Instructions.md
var CoderInstructions string

//go:embed titleGenerate.md
var TitleGenerationPrompt string
