package core

import _ "embed"

//go:embed Roles/coding.md
var CodingRole string

//go:embed Roles/documenting.md
var DocumentingRole string

//go:embed Roles/askAI.md
var AskAIRole string

//go:embed Instructions.md
var CoderInstructions string

//go:embed titleGenerate.md
var TitleGenerationPrompt string
