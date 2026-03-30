//go:build windows

package main

import "syscall"

func sysProcAttr() *syscall.SysProcAttr {
	// DETACHED_PROCESS = 0x00000008
	// CREATE_NEW_PROCESS_GROUP = 0x00000200
	return &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200}
}
