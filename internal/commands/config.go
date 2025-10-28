package commands

import (
	"fmt"
	"strings"
)

func init() {
	registerCommand("config", configCmd, nil)
}

func configCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()

	var b strings.Builder

	b.WriteString(fmt.Sprintf("appmode: %s\n", cfg.AppMode))
	b.WriteString("grpc:\n")
	b.WriteString(fmt.Sprintf("  addr: %s\n", cfg.GRPC.Addr))
	b.WriteString("generation:\n")
	b.WriteString(fmt.Sprintf("  modelcode: %s\n", cfg.Generation.ModelCode))
	b.WriteString(fmt.Sprintf("  temperature: %.1f\n", cfg.Generation.Temperature))
	b.WriteString(fmt.Sprintf("  topp: %.2f\n", cfg.Generation.TopP))
	b.WriteString(fmt.Sprintf("  topk: %.1f\n", cfg.Generation.TopK))
	b.WriteString(fmt.Sprintf("  outputlength: %d\n", cfg.Generation.OutputLength))
	b.WriteString("context:\n")

	if len(cfg.Context.Files) > 0 {
		b.WriteString("  files:\n")
		for _, f := range cfg.Context.Files {
			b.WriteString(fmt.Sprintf("    - %s\n", f))
		}
	} else {
		b.WriteString("  files: []\n")
	}

	if len(cfg.Context.Dirs) > 0 {
		b.WriteString("  dirs:\n")
		for _, d := range cfg.Context.Dirs {
			b.WriteString(fmt.Sprintf("    - %s\n", d))
		}
	} else {
		b.WriteString("  dirs: []\n")
	}

	if len(cfg.Context.Exclusions) > 0 {
		b.WriteString("  exclusions:\n")
		for _, e := range cfg.Context.Exclusions {
			b.WriteString(fmt.Sprintf("    - %s\n", e))
		}
	} else {
		b.WriteString("  exclusions: []\n")
	}

	return CommandOutput{Type: CommandResultString, Payload: strings.TrimSpace(b.String())}, true
}
