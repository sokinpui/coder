package commands

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/templates"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("template", templateCmd, "apply a prompt template", templateArgumentCompleter)
}

func templateArgumentCompleter(s SessionController, prefix string) []string {
	var names []string
	for _, t := range templates.Discover() {
		names = append(names, t.Name)
	}
	return names
}

func templateCmd(args string, s SessionController) (CommandOutput, bool) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: "Interactive command: use /template in TUI mode or specify template name: /template <name>",
		}, false
	}

	parts := strings.Fields(trimmed)
	name := parts[0]
	extra := strings.TrimSpace(trimmed[len(name):])

	tmpl, found := templates.Find(name)
	if !found {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: fmt.Sprintf("Template '%s' not found.", name),
		}, false
	}

	prompt := tmpl.Content
	if extra != "" {
		if strings.Contains(prompt, "{{PROMPT}}") {
			prompt = strings.ReplaceAll(prompt, "{{PROMPT}}", extra)
		} else if strings.Contains(prompt, "{{ARGS}}") {
			prompt = strings.ReplaceAll(prompt, "{{ARGS}}", extra)
		} else {
			prompt = prompt + "\n\n" + extra
		}
	}

	return CommandOutput{
		Type:         types.MessagesUpdated,
		Payload:      fmt.Sprintf("Applied template: %s", tmpl.Name),
		SubmitPrompt: prompt,
	}, true
}
