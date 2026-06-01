package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/itf"
	"strings"
)

func init() {
	registerCommand("itf", itfCmd, "apply code changes", nil)
}

type ItfResult struct {
	Summary       string
	AffectedFiles []string
	Raw           map[string][]string
	Success       bool
}

// Use itf to apply file operations
func ExecuteItf(content string, args string) ItfResult {
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
		return ItfResult{Summary: "Error applying changes: " + err.Error(), Success: false}
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
		return ItfResult{Summary: "No changes applied.", Raw: results, Success: true}
	}

	// Remove duplicates
	seen := make(map[string]struct{})
	uniqueFiles := []string{}
	for _, f := range affectedFiles {
		if _, ok := seen[f]; !ok {
			seen[f] = struct{}{}
			uniqueFiles = append(uniqueFiles, f)
		}
	}

	return ItfResult{
		Summary:       summary,
		AffectedFiles: uniqueFiles,
		Raw:           results,
		Success:       true,
	}
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

	res := ExecuteItf(lastAIResponse, args)
	s.SetLastModifiedFiles(res.AffectedFiles)

	if !res.Success {
		return CommandOutput{Type: types.MessagesUpdated, Payload: res.Summary}, false
	}

	// Mark that this session has applied changes
	if len(res.Raw["Created"]) > 0 ||
		len(res.Raw["Modified"]) > 0 ||
		len(res.Raw["Renamed"]) > 0 ||
		len(res.Raw["Deleted"]) > 0 {
		s.SetHasAppliedChanges(true)
	}

	// Update context paths based on itf results
	currentFiles := s.GetContextFiles()
	contextUpdated := false

	// Handle Created files
	if created := res.Raw["Created"]; len(created) > 0 {
		currentFiles = AppendUnique(currentFiles, created)
		contextUpdated = true
	}

	// Handle Deleted files
	if deleted := res.Raw["Deleted"]; len(deleted) > 0 {
		toRemove := make(map[string]struct{})
		for _, p := range deleted {
			toRemove[p] = struct{}{}
		}
		currentFiles = filterPaths(currentFiles, toRemove)
		contextUpdated = true
	}

	// Handle Renamed files
	if renamed := res.Raw["Renamed"]; len(renamed) > 0 {
		toRemove := make(map[string]struct{})
		var toAdd []string
		for _, entry := range renamed {
			parts := strings.Split(entry, " -> ")
			if len(parts) == 2 {
				toRemove[parts[0]] = struct{}{}
				toAdd = append(toAdd, parts[1])
			}
		}
		currentFiles = filterPaths(currentFiles, toRemove)
		currentFiles = AppendUnique(currentFiles, toAdd)
		contextUpdated = true
	}

	if contextUpdated {
		s.SetContextFiles(currentFiles)
		_ = s.LoadContext()
	}

	return CommandOutput{Type: types.MessagesUpdated, Payload: res.Summary}, res.Success
}
