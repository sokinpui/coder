package commands

import (
	"fmt"
	"github.com/sokinpui/coder/internal/types"
	"strings"
)

func init() {
	registerCommand("config", configCmd, "show current config", nil)
}

func configCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()

	var b strings.Builder
	fmt.Fprintf(&b, "server:\n  url: %s\n", cfg.Server.URL)
	fmt.Fprintf(&b, "generation:\n  modelcode: %s\n  streamdelay: %d\n",
		cfg.Generation.ModelCode, cfg.Generation.StreamDelay)

	fmt.Fprintf(&b, "clipboard:\n  copycmd: %s\n  pastecmd: %s\n", cfg.Clipboard.CopyCmd, cfg.Clipboard.PasteCmd)

	b.WriteString("config context:\n")
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
