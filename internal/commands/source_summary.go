package commands

import (
	"coder/internal/config"
	"fmt"
	"strings"
)

// formatContextSummary creates a vertical, indented list of source directories and files.
func formatContextSummary(context *config.Context) string {
	if len(context.Dirs) == 0 && len(context.Files) == 0 && len(context.Exclusions) == 0 {
		return ""
	}

	var b strings.Builder

	if len(context.Dirs) > 0 {
		b.WriteString("Directories:\n")
		for _, d := range context.Dirs {
			fmt.Fprintf(&b, "  %s\n", d)
		}
	}

	if len(context.Files) > 0 {
		b.WriteString("Files:\n")
		for _, f := range context.Files {
			fmt.Fprintf(&b, "  %s\n", f)
		}
	}

	if len(context.Exclusions) > 0 {
		b.WriteString("Exclusions:\n")
		for _, e := range context.Exclusions {
			fmt.Fprintf(&b, "  %s\n", e)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
