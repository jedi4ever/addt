//go:build windows
// +build windows

package main

import (
	"os"
)

// isatty checks if a file descriptor is a terminal (Windows version)
func isatty(fd int) bool {
	// On Windows, check if it's a character device
	var file *os.File
	switch fd {
	case 0:
		file = os.Stdin
	case 1:
		file = os.Stdout
	case 2:
		file = os.Stderr
	default:
		return false
	}

	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// GetTerminalSize returns the terminal dimensions (columns, lines)
// Windows version - returns default values
func GetTerminalSize() (int, int) {
	// TODO: Implement proper Windows terminal size detection using Windows API
	// For now, return reasonable defaults
	return 80, 24
}
