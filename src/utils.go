package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// tempDirs tracks temporary directories for cleanup
var tempDirs []string

// SetupCleanup sets up signal handlers for cleanup on exit
func SetupCleanup() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		Cleanup()
		os.Exit(1)
	}()
}

// Cleanup removes all temporary directories
func Cleanup() {
	for _, dir := range tempDirs {
		os.RemoveAll(dir)
	}
}

// IsTerminal checks if stdin and stdout are both terminals
func IsTerminal() bool {
	// Both stdin (0) and stdout (1) must be terminals for interactive mode
	// isatty() is implemented in platform-specific files (terminal_unix.go, terminal_windows.go)
	return isatty(0) && isatty(1)
}

// SafeCopyFile copies a file if it exists
func SafeCopyFile(src, dst string) {
	if _, err := os.Stat(src); err != nil {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	os.WriteFile(dst, data, 0600)
}

// LogCommand logs a command to the log file
func LogCommand(cfg *Config, cwd, containerName string, args []string) {
	f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] PWD: %s | Container: %s | Command: %s\n",
		timestamp, cwd, containerName, strings.Join(args, " "))
}
