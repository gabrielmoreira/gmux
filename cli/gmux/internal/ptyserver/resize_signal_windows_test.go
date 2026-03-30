//go:build windows

package ptyserver

import "os"

func notifyResizeSignal(ch chan<- os.Signal) func() {
	return func() {}
}
