package tools

import (
	"fmt"

	"github.comcom/sokinpui/pcat.go/pcat"
)

func init() {
	RegisterTool(
		Definition{
			ToolName:    "read_directories",
			Description: "Reads the content of directories given an array of paths.",
			Args: []ArgumentDefinition{
				{
					Name:        "paths",
					Type:        "array",
					Description: "An array of relative directory paths to read.",
				},
			},
		},
		readDirectories)
}

// readDirectories reads the content of one or more directories using pcat and returns the formatted output.
func readDirectories(args map[string]interface{}, lastAIResponse string) (string, error) {
	pathsArg, ok := args["paths"]
	if !ok {
		return "", fmt.Errorf("missing required argument: paths")
	}

	pathsInterface, ok := pathsArg.([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid type for argument 'paths': expected array of strings")
	}

	if len(pathsInterface) == 0 {
		return "No directories specified.", nil
	}

	var dirPaths []string
	for i, pathInterface := range pathsInterface {
		path, ok := pathInterface.(string)
		if !ok {
			return "", fmt.Errorf("invalid path at index %d: not a string", i)
		}
		dirPaths = append(dirPaths, path)
	}

	// Configure pcat to include line numbers and no headers.
	config := pcat.Config{
		NoHeader: true,
	}

	output, err := pcat.Read(dirPaths, config)
	if err != nil {
		return "", fmt.Errorf("failed to read directories with pcat: %w", err)
	}

	return output, nil
}
