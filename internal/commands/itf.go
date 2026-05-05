package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/itf"
	"strings"
)

func init() {
	registerCommand("itf", itfCmd, nil)
}

// Use itf to apply file operations
func ExecuteItf(content string, args string) (string, []string, bool) {
	fields := strings.Fields(args)
	config := itf.Config{}

	for _, arg := range fields {
		if strings.HasPrefix(arg, ".") {
			config.Extensions = append(config.Extensions, arg)
			continue
		}
		config.Files = append(config.Files, arg)
	}

	results, err := itf.Apply(content, config)
	if err != nil {
		return "Error applying changes: " + err.Error(), nil, false
	}

	var affectedFiles []string
	affectedFiles = append(affectedFiles, results["Created"]...)
	affectedFiles = append(affectedFiles, results["Modified"]...)

	for _, p := range results["Renamed"] {
		parts := strings.Split(p, " -> ")
		if len(parts) == 2 {
			affectedFiles = append(affectedFiles, parts[1])
		}
	}

	summary := itf.FormatResult(results)
	if summary == "" {
		return "No changes applied.", nil, true
	}

	// Remove duplicates
	seen := make(map[string]struct{})
	uniqueFiles := []string{}
	for _, f := range affectedFiles {
		if _, ok := seen[f]; !ok { seen[f] = struct{}{}; uniqueFiles = append(uniqueFiles, f) }
	}

	return summary, uniqueFiles, true
}

func itfCmd(args string, s SessionController) (CommandOutput, bool) {
	messages := s.GetMessages()
	var lastAIResponse string
	found := false
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == types.AIMessage {
			lastAIResponse = messages[i].Content
			found = true
			break
		}
	}

	if !found {
		return CommandOutput{Type: types.MessagesUpdated, Payload: "No AI response found to pipe to itf."}, false
	}

	result, affected, success := ExecuteItf(lastAIResponse, args)
	s.SetLastModifiedFiles(affected)
	return CommandOutput{Type: types.MessagesUpdated, Payload: result}, success
}
