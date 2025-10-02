package tools

import (
	"fmt"
	"strings"

	"github.com/sokinpui/itf.go/cli"
	"github.com/sokinpui/itf.go/itf"
)

func init() {
	RegisterTool(
		Definition{
			ToolName:    "upsert_files",
			Description: "Create or update files. This tool will automatically parse the content of your last response and apply the file changes.",
			Args: []ArgumentDefinition{
				{
					Name:        "paths",
					Type:        "array",
					Description: "An array of relative file paths you are creating or updating.",
				},
			},
		},
		upsertFiles,
	)
}

// upsertFiles creates or updates files by applying changes from the last AI response.
// The 'args' are ignored as itf.Apply parses the entire response content.
func upsertFiles(args map[string]interface{}, lastAIResponse string) (string, error) {
	config := cli.Config{} // Use default config

	summary, err := itf.Apply(lastAIResponse, config)
	if err != nil {
		return "", fmt.Errorf("failed to apply changes: %w", err)
	}

	var summaryBuilder strings.Builder
	summaryBuilder.WriteString("File upsert summary:\n")
	hasChanges := false
	for key, val := range summary {
		// itf.Apply returns map[string][]string
		if len(val) > 0 {
			hasChanges = true
			summaryBuilder.WriteString(fmt.Sprintf("  %s: %v\n", key, val))
		}
	}

	if !hasChanges {
		return "No file changes were applied.", nil
	}

	return strings.TrimSpace(summaryBuilder.String()), nil
}
