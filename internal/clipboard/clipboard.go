package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

type command struct {
	name string
	args []string
}

func writeToCmd(cmd command, text string) error {
	c := exec.Command(cmd.name, cmd.args...)
	c.Stdin = bytes.NewBufferString(text)
	return c.Run()
}

func Write(text string) error {
	if text == "" {
		return nil
	}

	var commands []command
	switch runtime.GOOS {
	case "darwin":
		commands = []command{{"pbcopy", nil}}
	case "linux":
		commands = []command{
			{"xclip", []string{"-selection", "clipboard"}},
			{"xsel", []string{"--clipboard", "--input"}},
		}
	case "windows":
		commands = []command{{"clip", nil}}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	var tried []string
	for _, cmd := range commands {
		tried = append(tried, cmd.name)
		if _, err := exec.LookPath(cmd.name); err != nil {
			continue
		}
		if err := writeToCmd(cmd, text); err == nil {
			return nil
		}
	}

	return fmt.Errorf("no clipboard tool found (tried %v)", tried)
}
