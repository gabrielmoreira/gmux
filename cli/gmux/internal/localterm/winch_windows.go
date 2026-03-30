//go:build windows

package localterm

import (
	"time"
)

// handleWinch listens for terminal resizes. Since Windows lacks SIGWINCH,
// we just poll occasionally. In a deeper integration we could read CONSOLE_WINDOW_BUFFER_SIZE_RECORD.
func (a *Attach) handleWinch() {
	var lastCols, lastRows uint16
	if cols, rows, err := TerminalSize(); err == nil {
		lastCols, lastRows = cols, rows
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.done:
			return
		case <-ticker.C:
			if cols, rows, err := TerminalSize(); err == nil {
				if cols != lastCols || rows != lastRows {
					lastCols, lastRows = cols, rows
					a.resizeFn(cols, rows)
				}
			}
		}
	}
}
