package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/types"
)

func RegisterShellCommands(cfg *config.Config) []string {
	var errors []string
	if cfg.ShellCommands == nil {
		return nil
	}

	for name, cmdDef := range cfg.ShellCommands {
		if IsBuiltIn(name) {
			errors = append(errors, fmt.Sprintf("Shell command '%s' conflicts with a built-in command.", name))
			continue
		}

		definition := cmdDef // capture for closure
		commandName := name

		registerCommand(commandName, func(args string, s SessionController) (CommandOutput, bool) {
			execStr := definition.Exec
			argList := strings.Fields(args)

			// Simple positional argument replacement $1, $2, etc.
			for i, arg := range argList {
				placeholder := fmt.Sprintf("$%d", i+1)
				execStr = strings.ReplaceAll(execStr, placeholder, arg)
			}

			// Clean up remaining placeholders if any
			// We don't use a regex for simplicity, assuming $N format.
			for i := len(argList); i < 10; i++ {
				placeholder := fmt.Sprintf("$%d", i+1)
				if strings.Contains(execStr, placeholder) {
					execStr = strings.ReplaceAll(execStr, placeholder, "")
				}
			}

			out, err := exec.Command("sh", "-c", execStr).CombinedOutput()

			outputType := types.MessagesUpdated
			payload := string(out)
			success := true

			if err != nil {
				// If command failed, we still treat it as a result but with error info
				if payload == "" {
					payload = err.Error()
				}
				success = false
			}

			return CommandOutput{
				Type:    outputType,
				Payload: payload,
				Metadata: map[string]any{
					"canAISee": definition.CanAISee,
					"isShell":  true,
				},
			}, success
		}, definition.Description, nil)
	}

	return errors
}
