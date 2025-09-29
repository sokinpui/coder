package core

import _ "embed"

//go:embed Roles/coding.txt
var CodingRole string

//go:embed Roles/documenting.txt
var DocumentingRole string

//go:embed Roles/agent.txt
var AgentRole string

//go:embed Roles/askAI.txt
var AskAIRole string

//go:embed Instructions.txt
var CoderInstructions string

//go:embed titleGenerate.txt
var TitleGenerationPrompt string
