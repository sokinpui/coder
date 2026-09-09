package commands

import (
	"fmt"
	"strings"
)

func formatFileListSummary(files []string, documents []string) string {
	if len(files) == 0 && len(documents) == 0 {
		return ""
	}

	var b strings.Builder
	if len(files) > 0 {
		b.WriteString("Files:\n")
		for _, f := range files {
			fmt.Fprintf(&b, "  %s\n", f)
		}
	}
	if len(documents) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Documents:\n")
		for _, d := range documents {
			fmt.Fprintf(&b, "  %s\n", d)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
