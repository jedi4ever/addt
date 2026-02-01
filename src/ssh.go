package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HandleSSHForwarding configures SSH forwarding based on config
func HandleSSHForwarding(cfg *Config, homeDir, username string) []string {
	var args []string

	if cfg.SSHForward == "agent" || cfg.SSHForward == "true" {
		sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
		if sshAuthSock != "" {
			// Check if socket exists and is accessible
			if _, err := os.Stat(sshAuthSock); err == nil {
				// Check for macOS launchd sockets (won't work)
				if strings.Contains(sshAuthSock, "com.apple.launchd") || strings.Contains(sshAuthSock, "/var/folders/") {
					fmt.Println("Warning: SSH agent forwarding not supported on macOS (use DCLAUDE_SSH_FORWARD=keys)")
				} else {
					args = append(args, "-v", fmt.Sprintf("%s:/ssh-agent", sshAuthSock))
					args = append(args, "-e", "SSH_AUTH_SOCK=/ssh-agent")

					// Mount safe SSH files only
					sshDir := filepath.Join(homeDir, ".ssh")
					if _, err := os.Stat(sshDir); err == nil {
						tmpDir, err := os.MkdirTemp("", "ssh-safe-*")
						if err == nil {
							tempDirs = append(tempDirs, tmpDir)

							// Copy safe files
							SafeCopyFile(filepath.Join(sshDir, "config"), filepath.Join(tmpDir, "config"))
							SafeCopyFile(filepath.Join(sshDir, "known_hosts"), filepath.Join(tmpDir, "known_hosts"))

							// Copy public keys
							files, _ := filepath.Glob(filepath.Join(sshDir, "*.pub"))
							for _, f := range files {
								SafeCopyFile(f, filepath.Join(tmpDir, filepath.Base(f)))
							}

							args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.ssh:ro", tmpDir, username))
						}
					}
				}
			}
		}
	} else if cfg.SSHForward == "keys" {
		sshDir := filepath.Join(homeDir, ".ssh")
		if _, err := os.Stat(sshDir); err == nil {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.ssh:ro", sshDir, username))
		}
	}

	return args
}
