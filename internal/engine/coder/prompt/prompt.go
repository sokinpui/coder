package prompt

import _ "embed"

//go:embed Instructions.md
var Instructions string

//go:embed titleGenerate.md
var TitleGenerationPrompt string
