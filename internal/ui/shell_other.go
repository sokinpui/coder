//go:build !unix

package ui

import (
	"io"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type plainExecCmd struct {
	cmd *exec.Cmd
}

func wrapInteractiveShellCmd(cmd *exec.Cmd) tea.ExecCommand {
	return &plainExecCmd{cmd: cmd}
}

func (c *plainExecCmd) SetStdin(r io.Reader)  { c.cmd.Stdin = r }
func (c *plainExecCmd) SetStdout(w io.Writer) { c.cmd.Stdout = w }
func (c *plainExecCmd) SetStderr(w io.Writer) { c.cmd.Stderr = w }
func (c *plainExecCmd) Run() error            { return c.cmd.Run() }
