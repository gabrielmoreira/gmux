//go:build windows

package ptyserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"

	gopty "github.com/aymanbagabas/go-pty"
)

// PTY handle to allow resizing
type winPty struct {
	pty gopty.Pty
}

func (w *winPty) Read(p []byte) (n int, err error) {
	return w.pty.Read(p)
}

func (w *winPty) Write(p []byte) (n int, err error) {
	return w.pty.Write(p)
}

func (w *winPty) Close() error {
	return w.pty.Close()
}

type ptyProcess struct {
	cmd *gopty.Cmd
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
	if p.cmd != nil && p.cmd.ProcessState != nil {
		return p.cmd.ProcessState.ExitCode()
	}
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
	return p.cmd.Process.Kill()
}

func (p *ptyProcess) KillProcessGroup() error {
	if p.cmd.Process == nil {
		return nil
	}
	// Windows doesn't have process groups natively mapped to signals.
	// Terminate the process cleanly if possible, otherwise hard kill.
	// Since gmux creates new process groups we might need deeper handling
	// in the future but returning Kill is safe enough.
	return p.cmd.Process.Kill()
}

func (p *ptyProcess) SignalWinch() {
	// Not applicable on Windows. Go-pty sends the resize directly through the conpty API.
}

func startPTY(path string, args, env []string, dir string, cols, rows uint16) (io.ReadWriteCloser, PtyProcess, error) {
	resolvedPath, err := exec.LookPath(path)
	if err != nil {
		resolvedPath = path // fallback to original if not found in PATH
	}

	pt, err := gopty.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create pty: %w", err)
	}

	if err := pt.Resize(int(cols), int(rows)); err != nil {
		// Ignore resize errors before start
	}

	ptyCmd := pt.CommandContext(context.Background(), resolvedPath, commandArgs(args)...)
	ptyCmd.Dir = dir
	ptyCmd.Env = env

	if err := ptyCmd.Start(); err != nil {
		pt.Close()
		return nil, nil, err
	}

	return &winPty{pty: pt}, &ptyProcess{cmd: ptyCmd}, nil
}

func resizePTY(ptmx io.ReadWriteCloser, cols, rows, x, y uint16) error {
	if w, ok := ptmx.(*winPty); ok && w != nil && w.pty != nil {
		return w.pty.Resize(int(cols), int(rows))
	}
	return nil
}

func listenUnix(addr string) (net.Listener, error) {
	return net.Listen("unix", addr)
}
