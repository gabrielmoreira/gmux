//go:build !windows

package ptyserver

import (
	"io"
	"net"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

type ptyProcess struct {
	cmd *exec.Cmd
}

func (p *ptyProcess) Pid() int {
	if p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *ptyProcess) Wait() error {
	return p.cmd.Wait()
}

func (p *ptyProcess) ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func (p *ptyProcess) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
}

func (p *ptyProcess) KillProcessGroup() error {
	if p.cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
}

func (p *ptyProcess) SignalWinch() {
	if p.cmd.Process == nil {
		return
	}
	syscall.Kill(-p.cmd.Process.Pid, syscall.SIGWINCH)
}

func startPTY(path string, args, env []string, dir string, cols, rows uint16) (io.ReadWriteCloser, PtyProcess, error) {
	cmd := exec.Command(path, commandArgs(args)...)
	cmd.Dir = dir
	cmd.Env = env

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
	if err != nil {
		return nil, nil, err
	}

	return ptmx, &ptyProcess{cmd: cmd}, nil
}

func resizePTY(ptmx io.ReadWriteCloser, cols, rows, x, y uint16) error {
	f, ok := ptmx.(*os.File)
	if !ok {
		return nil
	}
	return pty.Setsize(f, &pty.Winsize{
		Cols: cols,
		Rows: rows,
		X:    x,
		Y:    y,
	})
}

func listenUnix(addr string) (net.Listener, error) {
	oldUmask := syscall.Umask(0o077)
	defer syscall.Umask(oldUmask)
	return net.Listen("unix", addr)
}
