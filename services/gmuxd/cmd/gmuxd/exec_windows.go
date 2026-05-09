//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS=0x08
	}
}

// defaultShell returns the first executable shell candidate for Windows,
// preferring SHELL when it points to a runnable program.
func defaultShell() string {
	if shell := executableShell(os.Getenv("SHELL")); shell != "" {
		return shell
	}
	if comspec := executableShell(os.Getenv("COMSPEC")); comspec != "" {
		return comspec
	}
	return "cmd.exe"
}

func executableShell(candidate string) string {
	if candidate == "" {
		return ""
	}
	path, err := exec.LookPath(candidate)
	if err != nil {
		return ""
	}
	return path
}
