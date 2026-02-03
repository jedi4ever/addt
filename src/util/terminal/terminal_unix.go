//go:build linux || darwin
// +build linux darwin

package terminal

import (
	"golang.org/x/sys/unix"
)

// isatty checks if a file descriptor is a terminal (cross-platform)
func isatty(fd int) bool {
	_, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	return err == nil
}

// GetTerminalSize returns the terminal dimensions (columns, lines)
func GetTerminalSize() (int, int) {
	ws, err := unix.IoctlGetWinsize(0, unix.TIOCGWINSZ)
	if err != nil {
		return 80, 24 // Default fallback
	}
	return int(ws.Col), int(ws.Row)
}
