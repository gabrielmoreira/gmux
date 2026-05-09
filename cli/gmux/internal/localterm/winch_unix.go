//go:build !windows

package localterm

import (
	"os"
	"os/signal"
	"syscall"
)

// handleWinch listens for SIGWINCH and forwards terminal size to the PTY.
func (a *Attach) handleWinch() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	defer signal.Stop(ch)

	for {
		select {
		case <-a.done:
			return
		case <-ch:
			if cols, rows, err := TerminalSize(); err == nil {
				a.resizeFn(cols, rows)
			}
		}
	}
}
