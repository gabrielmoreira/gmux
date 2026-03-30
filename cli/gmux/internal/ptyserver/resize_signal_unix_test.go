//go:build !windows

package ptyserver

import (
	"os"
	"os/signal"
	"syscall"
)

func notifyResizeSignal(ch chan<- os.Signal) func() {
	signal.Notify(ch, syscall.SIGWINCH)
	return func() {
		signal.Stop(ch)
	}
}
