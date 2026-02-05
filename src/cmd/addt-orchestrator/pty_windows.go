//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// startPty starts a command with a PTY (Windows stub)
func startPty(cmd *exec.Cmd) (*os.File, error) {
	return nil, fmt.Errorf("PTY not supported on Windows")
}
