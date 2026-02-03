package terminal

// IsTerminal checks if stdin and stdout are both terminals
func IsTerminal() bool {
	// Both stdin (0) and stdout (1) must be terminals for interactive mode
	// isatty() is implemented in platform-specific files (terminal_unix.go, terminal_windows.go)
	return isatty(0) && isatty(1)
}
