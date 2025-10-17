package commands

import (
	"coder/internal/config"
	"fmt"
	"strings"
)

// formatSourceSummary creates a vertical, indented list of source directories and files.
func formatSourceSummary(sources *config.FileSources) string {
	if len(sources.Dirs) == 0 && len(sources.Files) == 0 {
		return ""
	}

	var b strings.Builder

	if len(sources.Dirs) > 0 {
		b.WriteString("Directories:\n")
		for _, d := range sources.Dirs {
			fmt.Fprintf(&b, "  %s\n", d)
		}
	}

	if len(sources.Files) > 0 {
		b.WriteString("Files:\n")
		for _, f := range sources.Files {
			fmt.Fprintf(&b, "  %s\n", f)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}
