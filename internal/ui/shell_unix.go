//go:build unix

package ui

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sys/unix"
)

type ttySafeExecCmd struct {
	cmd *exec.Cmd
}

func wrapInteractiveShellCmd(cmd *exec.Cmd) tea.ExecCommand {
	return &ttySafeExecCmd{cmd: cmd}
}

func (c *ttySafeExecCmd) SetStdin(r io.Reader)  { c.cmd.Stdin = r }
func (c *ttySafeExecCmd) SetStdout(w io.Writer) { c.cmd.Stdout = w }
func (c *ttySafeExecCmd) SetStderr(w io.Writer) { c.cmd.Stderr = w }

func (c *ttySafeExecCmd) Run() error {
	pgrp := unix.Getpgrp()
	err := c.cmd.Run()

	if f, ok := c.cmd.Stdin.(*os.File); ok {
		restoreForegroundPgrp(int(f.Fd()), pgrp)
		return err
	}

	if tty, ttyErr := os.OpenFile("/dev/tty", os.O_RDWR, 0); ttyErr == nil {
		restoreForegroundPgrp(int(tty.Fd()), pgrp)
		_ = tty.Close()
	}
	return err
}

func restoreForegroundPgrp(fd int, pgrp int) {
	curFg, err := unix.IoctlGetInt(fd, unix.TIOCGPGRP)
	if err != nil || curFg == pgrp {
		return
	}
	signal.Ignore(syscall.SIGTTOU, syscall.SIGTTIN)
	_ = unix.IoctlSetPointerInt(fd, unix.TIOCSPGRP, pgrp)
	signal.Reset(syscall.SIGTTOU, syscall.SIGTTIN)
}
