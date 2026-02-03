package util

import (
	"os"
	"os/signal"
	"syscall"
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
