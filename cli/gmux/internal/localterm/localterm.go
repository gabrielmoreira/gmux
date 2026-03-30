// Package localterm provides transparent local terminal attach for gmux.
// When stdin is an interactive terminal, it relays I/O between the calling
// terminal and the child's PTY, making "gmux app" behave like "app".
package localterm

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"sync"

	"golang.org/x/term"
)

// Attach represents a local terminal attachment to a PTY session.
// It relays stdin→PTY and PTY output→stdout, handles SIGWINCH,
// and can be detached without killing the session.
type Attach struct {
	stdin  *os.File
	stdout *os.File

	ptyWriter     io.Writer
	resizeFn      func(cols, rows uint16)
	oldState      *term.State
	stdinMode     uint32
	hasStdinMode  bool
	stdoutMode    uint32
	hasStdoutMode bool
	mu            sync.Mutex
	detached      bool
	done          chan struct{}
}

// IsInteractive returns true if stdin is a terminal.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// TerminalSize returns the current terminal dimensions.
// Windows needs a stdout fallback because detached consoles may leave stdin
// non-terminal; Unix stays stdin-first so sizing follows the controlling TTY.
func TerminalSize() (cols, rows uint16, err error) {
	files := []*os.File{os.Stdin, os.Stdout}
	if runtime.GOOS == "windows" {
		files = []*os.File{os.Stdout, os.Stdin}
	}
	for _, f := range files {
		w, h, err := term.GetSize(int(f.Fd()))
		if err == nil {
			return uint16(w), uint16(h), nil
		}
	}
	return 0, 0, err
}

// Config for creating a local terminal attachment.
type Config struct {
	// PTYWriter receives bytes from stdin (writes to ptmx).
	PTYWriter io.Writer
	// ResizeFn is called when the terminal is resized.
	ResizeFn func(cols, rows uint16)
}

// New creates a local terminal attachment. It puts the terminal in raw
// mode and enables focus reporting, but does NOT begin relaying I/O yet.
// Call Start() after the PTY writer and resize function are safe to
// invoke, and Detach() to restore the terminal.
//
// Splitting construction from I/O startup lets callers set up a PTY
// server that already knows about this Attach (as its LocalOut) before
// any PTY bytes are read, so fast-exiting commands don't race the
// attach wiring and lose their output.
func New(cfg Config) (*Attach, error) {
	stdin := os.Stdin
	stdout := os.Stdout

	stdoutMode, stdoutErr := enableVT(stdout)
	stdinMode, stdinErr := enableVTInput(stdin)

	// Enter raw mode — keystrokes pass through to child
	oldState, err := term.MakeRaw(int(stdin.Fd()))
	if err != nil {
		if stdinErr == nil {
			restoreVTInput(stdin, stdinMode)
		}
		if stdoutErr == nil {
			restoreVT(stdout, stdoutMode)
		}
		return nil, err
	}

	a := &Attach{
		stdin:         stdin,
		stdout:        stdout,
		ptyWriter:     cfg.PTYWriter,
		resizeFn:      cfg.ResizeFn,
		oldState:      oldState,
		stdinMode:     stdinMode,
		hasStdinMode:  stdinErr == nil,
		stdoutMode:    stdoutMode,
		hasStdoutMode: stdoutErr == nil,
		done:          make(chan struct{}),
	}

	// Enable focus reporting so we can resize the PTY when the host
	// terminal gains focus (the terminal's size is authoritative at
	// that point, even if a browser client resized the PTY earlier).
	stdout.WriteString("\x1b[?1004h")

	return a, nil
}

// Start begins relaying stdin→PTY and handling SIGWINCH. Call once,
// after New, when the configured PTYWriter and ResizeFn are ready to
// be invoked.
func (a *Attach) Start() {
	go a.readStdin()
	go a.handleWinch()
}

// Write sends PTY output to the local terminal's stdout.
// Called by the ptyserver broadcast loop. Safe to call after Detach (no-op).
func (a *Attach) Write(p []byte) (int, error) {
	a.mu.Lock()
	if a.detached {
		a.mu.Unlock()
		return len(p), nil // swallow silently
	}
	a.mu.Unlock()
	return a.stdout.Write(p)
}

// Detach disconnects the local terminal without stopping the session.
// Restores terminal state. Safe to call multiple times.
func (a *Attach) Detach() {
	a.mu.Lock()
	if a.detached {
		a.mu.Unlock()
		return
	}
	a.detached = true
	a.mu.Unlock()

	// Soft reset the host terminal before restoring modes.
	// This clears Alternate Screen, Bracketed Paste, SGR Mouse Reporting,
	// and focus reporting which orphaned TUI processes may have left active.
	if term.IsTerminal(int(a.stdout.Fd())) {
		a.stdout.Write([]byte("\x1b[?1004l\x1b[?1049l\x1b[?1006l\x1b[?1003l\x1b[?1002l\x1b[?1000l\x1b[?2004l\x1b[?25h\x1b[m"))
	}
	if a.oldState != nil {
		term.Restore(int(a.stdin.Fd()), a.oldState)
	}
	if a.hasStdinMode {
		restoreVTInput(a.stdin, a.stdinMode)
	}
	if a.hasStdoutMode {
		restoreVT(a.stdout, a.stdoutMode)
	}
	close(a.done)
}

// Detached returns true if the local terminal has been detached.
func (a *Attach) Detached() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.detached
}

// Done returns a channel closed when the local attach is detached
// (either explicitly or because stdin closed).
func (a *Attach) Done() <-chan struct{} {
	return a.done
}

// focusIn is the escape sequence terminals send when they gain focus
// (CSI ? 1004 h enables reporting; the terminal sends ESC [I on focus).
var focusIn = []byte("\x1b[I")

// termSize returns the current dimensions of the attached terminal.
func (a *Attach) termSize() (cols, rows uint16, err error) {
	return TerminalSize()
}

// readStdin reads from the calling terminal and writes to the PTY.
func (a *Attach) readStdin() {
	buf := make([]byte, 4096)
	for {
		n, err := a.stdin.Read(buf)
		if n > 0 {
			a.mu.Lock()
			detached := a.detached
			a.mu.Unlock()
			if detached {
				return
			}
			data := buf[:n]
			// When the host terminal gains focus, re-assert its size.
			// A browser client may have resized the PTY to its viewport;
			// focus means the user is back in the native terminal.
			if bytes.Contains(data, focusIn) {
				if cols, rows, err := a.termSize(); err == nil {
					a.resizeFn(cols, rows)
				}
			}
			a.ptyWriter.Write(data)
		}
		if err != nil {
			// stdin closed (terminal gone) — detach gracefully
			a.Detach()
			return
		}
	}
}
