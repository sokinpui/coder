package prompt

import _ "embed"

//go:embed Instructions.md
var Instructions string

//go:embed webfetchExtract.md
var WebFetchExtractPrompt string
