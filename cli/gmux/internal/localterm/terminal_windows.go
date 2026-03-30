//go:build windows

package localterm

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT enables virtual terminal processing on Windows console output.
// Returns the previous mode so it can be restored on exit.
func enableVT(f *os.File) (uint32, error) {
	var mode uint32
	err := windows.GetConsoleMode(windows.Handle(f.Fd()), &mode)
	if err != nil {
		return 0, err
	}
	newMode := mode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.DISABLE_NEWLINE_AUTO_RETURN
	err = windows.SetConsoleMode(windows.Handle(f.Fd()), newMode)
	return mode, err
}

func restoreVT(f *os.File, mode uint32) error {
	return windows.SetConsoleMode(windows.Handle(f.Fd()), mode)
}

func enableVTInput(f *os.File) (uint32, error) {
	var mode uint32
	err := windows.GetConsoleMode(windows.Handle(f.Fd()), &mode)
	if err != nil {
		return 0, err
	}
	newMode := mode | windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	err = windows.SetConsoleMode(windows.Handle(f.Fd()), newMode)
	return mode, err
}

func restoreVTInput(f *os.File, mode uint32) error {
	return windows.SetConsoleMode(windows.Handle(f.Fd()), mode)
}
