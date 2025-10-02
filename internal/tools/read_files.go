package tools

import (
	"fmt"

	"github.com/sokinpui/pcat.go/pcat"
)

func init() {
	RegisterTool(
		Definition{
			ToolName:    "read_files",
			Description: "Reads the content of files given an array of path.",
			Args: []ArgumentDefinition{
				{
					Name:        "paths",
					Type:        "array",
					Description: "An array of relative file paths to read.",
				},
			},
		},
		readFiles)
}

// readFiles reads the content of one or more files using pcat and returns the formatted output.
func readFiles(args map[string]interface{}) (string, error) {
	pathsArg, ok := args["paths"]
	if !ok {
		return "", fmt.Errorf("missing required argument: paths")
	}

	pathsInterface, ok := pathsArg.([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid type for argument 'paths': expected array of strings")
	}

	if len(pathsInterface) == 0 {
		return "No files specified.", nil
	}

	var filePaths []string
	for i, pathInterface := range pathsInterface {
		path, ok := pathInterface.(string)
		if !ok {
			return "", fmt.Errorf("invalid path at index %d: not a string", i)
		}
		filePaths = append(filePaths, path)
	}

	// Configure pcat to include line numbers and no headers.
	config := pcat.Config{
		NoHeader: true,
	}

	output, err := pcat.Read(filePaths, config)
	if err != nil {
		return "", fmt.Errorf("failed to read files with pcat: %w", err)
	}

	return output, nil
}
