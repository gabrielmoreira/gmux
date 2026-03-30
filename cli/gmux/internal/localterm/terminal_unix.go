//go:build !windows

package localterm

import "os"

func enableVT(f *os.File) (uint32, error) {
	return 0, nil
}

func restoreVT(f *os.File, mode uint32) error {
	return nil
}

func enableVTInput(f *os.File) (uint32, error) {
	return 0, nil
}

func restoreVTInput(f *os.File, mode uint32) error {
	return nil
}
