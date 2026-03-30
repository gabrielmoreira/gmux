//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultShellUsesExecutableShellEnv(t *testing.T) {
	t.Setenv("SHELL", "cmd.exe")
	t.Setenv("COMSPEC", "")

	got := defaultShell()
	if !strings.EqualFold(filepath.Base(got), "cmd.exe") {
		t.Fatalf("defaultShell() = %q, want cmd.exe from SHELL", got)
	}
}

func TestDefaultShellFallsBackWhenShellEnvIsNotExecutable(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/bash")
	t.Setenv("COMSPEC", "cmd.exe")

	got := defaultShell()
	if !strings.EqualFold(filepath.Base(got), "cmd.exe") {
		t.Fatalf("defaultShell() = %q, want cmd.exe fallback", got)
	}
}

func TestDefaultShellFallsBackToCmdExeWhenEnvIsEmpty(t *testing.T) {
	t.Setenv("SHELL", "")
	t.Setenv("COMSPEC", "")

	if got := defaultShell(); got != "cmd.exe" {
		t.Fatalf("defaultShell() = %q, want %q", got, "cmd.exe")
	}
}
