package commands

import (
	"fmt"
	"strings"
)

func formatFileListSummary(files []string) string {
	if len(files) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Files:\n")
	for _, f := range files {
		fmt.Fprintf(&b, "  %s\n", f)
	}
	return strings.TrimRight(b.String(), "\n")
}
