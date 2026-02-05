//go:build !windows

package main

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// startPty starts a command with a PTY
func startPty(cmd *exec.Cmd) (*os.File, error) {
	return pty.Start(cmd)
}
