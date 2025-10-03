package core

import _ "embed"

//go:embed Roles/coding.md
var CodingRole string

//go:embed Roles/documenting.md
var DocumentingRole string

//go:embed Roles/agent_main.md
var AgentRole string

//go:embed Roles/agent_coding.md
var AgentCodingRole string

//go:embed Roles/agent_writing.md
var AgentWritingRole string

//go:embed Roles/agent_general.md
var AgentGeneralRole string

//go:embed Roles/askAI.md
var AskAIRole string

//go:embed Instructions.md
var CoderInstructions string

//go:embed titleGenerate.md
var TitleGenerationPrompt string

//go:embed tool-call.txt
var ToolCallPrompt string
