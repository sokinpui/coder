package commands

import (
	"fmt"
	"strings"

	"github.com/sokinpui/coder/internal/types"
)

func init() {
	registerCommand("image", imageCmd, "toggle Kitty graphics image display in terminal", imageArgumentCompleter)
}

func imageArgumentCompleter(s SessionController, prefix string) []string {
	return []string{"on", "off", "toggle"}
}

func imageCmd(args string, s SessionController) (CommandOutput, bool) {
	cfg := s.GetConfig()
	trimmed := strings.ToLower(strings.TrimSpace(args))
	switch trimmed {
	case "on", "enable", "true", "1":
		cfg.KittyImage = true
	case "off", "disable", "false", "0":
		cfg.KittyImage = false
	default:
		cfg.KittyImage = !cfg.KittyImage
	}

	state := "disabled"
	if cfg.KittyImage {
		state = "enabled"
	}
	return CommandOutput{
		Type:    types.MessagesUpdated,
		Payload: fmt.Sprintf("Kitty image display %s.", state),
	}, true
}
