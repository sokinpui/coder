package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("config", configCmd, nil)
}

func configCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("server:\n  addr: %s\n", cfg.Server.Addr))
	b.WriteString(fmt.Sprintf("generation:\n  modelcode: %s\n  temperature: %.1f\n  topp: %.2f\n  topk: %.1f\n  outputlength: %d\n  streamdelay: %d\n",
		cfg.Generation.ModelCode, cfg.Generation.Temperature, cfg.Generation.TopP, cfg.Generation.TopK, cfg.Generation.OutputLength, cfg.Generation.StreamDelay))

	b.WriteString(fmt.Sprintf("clipboard:\n  copycmd: %s\n  pastecmd: %s\n", cfg.Clipboard.CopyCmd, cfg.Clipboard.PasteCmd))

	b.WriteString("context:\n")
	writeList(&b, "files", cfg.Context.Files)
	writeList(&b, "dirs", cfg.Context.Dirs)
	writeList(&b, "exclusions", cfg.Context.Exclusions)

	fmt.Fprintf(&b, "ui:\n  markdowntheme: %s\n", cfg.UI.MarkdownTheme)

	return CommandOutput{Type: types.MessagesUpdated, Payload: strings.TrimSpace(b.String())}, true
}

func writeList(b *strings.Builder, label string, items []string) {
	if len(items) == 0 {
		fmt.Fprintf(b, "  %s: []\n", label)
		return
	}
	fmt.Fprintf(b, "  %s:\n", label)
	for _, item := range items {
		fmt.Fprintf(b, "    - %s\n", item)
	}
}
