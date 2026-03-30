//go:build windows

package main

import (
	"os"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS=0x08
	}
}

// defaultShell returns the user's preferred shell, defaulting to /bin/sh.
func defaultShell() string {
	shell := os.Getenv("SHELL") // If running in mintty or something similar
	if shell != "" {
		return shell
	}
	// Fallbacks for Windows native env
	comspec := os.Getenv("COMSPEC")
	if comspec != "" {
		return comspec
	}
	return "cmd.exe"
}
