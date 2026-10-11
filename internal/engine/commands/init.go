package commands

import (
	"github.com/sokinpui/coder/internal/engine"
	"github.com/sokinpui/coder/internal/engine/agent/skills"
	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("init", initCmd, "initialize or update AGENTS.md guide", nil)
}

func initCmd(args string, s SessionController) (CommandOutput, bool) {
	if !s.Capabilities().Has(engine.CapToolLoop) {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: "/init requires agent mode. Switch using /mode agent or start with /new agent.",
		}, false
	}

	reader, ok := s.(engine.AgentFileReader)
	if !ok {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: "Agent file reader not supported in this session.",
		}, false
	}

	skill, found := skills.Find("init")
	if !found {
		return CommandOutput{
			Type:    types.MessagesUpdated,
			Payload: "Init skill not found.",
		}, false
	}

	readOut, ok := reader.ReadAgentFiles("/init", []string{skill.Path})
	if !ok {
		return readOut, false
	}

	prompt := "Please follow the instructions in the init skill above to inspect the repository and create or update AGENTS.md."
	if args != "" {
		prompt += "\nAdditional user instructions: " + args
	}

	return CommandOutput{
		Type:            types.MessagesUpdated,
		Payload:         readOut.Payload,
		SubmitPrompt:    prompt,
		IsAgentFileRead: true,
	}, true
}
