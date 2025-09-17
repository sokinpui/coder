package utils

import "regexp"

// ansiRegex matches ANSI escape codes.
// This is used to strip terminal control sequences from command output
// before sending it to the web UI, which cannot render them.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

// StripAnsi removes ANSI escape codes from a string.
func StripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
