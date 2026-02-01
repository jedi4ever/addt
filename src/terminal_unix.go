//go:build linux || darwin
// +build linux darwin

package main

import (
	"golang.org/x/sys/unix"
)

// isatty checks if a file descriptor is a terminal (cross-platform)
func isatty(fd int) bool {
	_, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	return err == nil
}
