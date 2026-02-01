package commands

import (
	"github.com/sokinpui/coder/internal/types"
	"github.com/sokinpui/itf"
	"strings"
)

func init() {
	registerCommand("itf", itfCmd, nil)
}

// ExecuteItf runs the 'itf' command with the given content as stdin.
func ExecuteItf(content string, args string) (string, bool) {
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
		return "Error applying changes: " + err.Error(), false
	}

	summary := itf.FormatResult(results)
	if summary == "" {
		return "No changes applied.", true
	}

	return summary, true
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

	result, success := ExecuteItf(lastAIResponse, args)
	return CommandOutput{Type: types.MessagesUpdated, Payload: result}, success
}
