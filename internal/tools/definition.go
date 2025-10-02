package tools

import (
	"encoding/json"
	"fmt"
	"sort"
)

// ArgumentDefinition defines a single argument for a tool.
type ArgumentDefinition struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Definition represents the definition of a tool that can be presented to the AI model.
type Definition struct {
	ToolName    string               `json:"tool"`
	Description string               `json:"description"`
	Args        []ArgumentDefinition `json:"args,omitempty"`
}

// Tool represents a registered tool with its function and definition.
type Tool struct {
	Function   ToolFunc
	Definition Definition
}

var registry = make(map[string]Tool)

// RegisterTool adds a new tool to the registry. This is not thread-safe
// and should be called from `init()` functions only.
func RegisterTool(definition Definition, fn ToolFunc) {
	name := definition.ToolName
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("tool with name '%s' is already registered", name))
	}
	registry[name] = Tool{
		Function:   fn,
		Definition: definition,
	}
}

// GetToolDefinitions returns a slice of all registered tool definitions, sorted by name.
func GetToolDefinitions() []Definition {
	defs := make([]Definition, 0, len(registry))
	for _, tool := range registry {
		defs = append(defs, tool.Definition)
	}

	// Sort for consistent output
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].ToolName < defs[j].ToolName
	})

	return defs
}

// GenerateToolDocsJSON generates a JSON string of all tool definitions.
func GenerateToolDocsJSON() (string, error) {
	defs := GetToolDefinitions()
	bytes, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// GetToolDefinitionsByNames returns a slice of tool definitions for the given names.
func GetToolDefinitionsByNames(names []string) []Definition {
	defs := make([]Definition, 0, len(names))
	for _, name := range names {
		if tool, ok := registry[name]; ok {
			defs = append(defs, tool.Definition)
		}
	}

	// Sort for consistent output
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].ToolName < defs[j].ToolName
	})

	return defs
}

// GenerateToolDocsJSONForTools generates a JSON string of specified tool definitions.
func GenerateToolDocsJSONForTools(toolNames []string) (string, error) {
	defs := GetToolDefinitionsByNames(toolNames)
	if len(defs) == 0 {
		return "", nil
	}
	bytes, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
