package docker

import (
	"fmt"
	"os"
	"strings"
)

// handleSSHAgentForwarding forwards the SSH agent socket into the container
func (p *DockerProvider) handleSSHAgentForwarding(homeDir, username string) []string {
	var args []string

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return args
	}

	// Check if socket exists and is accessible
	if _, err := os.Stat(sshAuthSock); err != nil {
		return args
	}

	// Check for macOS launchd sockets (won't work in Docker)
	if strings.Contains(sshAuthSock, "com.apple.launchd") || strings.Contains(sshAuthSock, "/var/folders/") {
		fmt.Println("Warning: SSH agent forwarding not supported on macOS (use ADDT_SSH_FORWARD_MODE=keys)")
		return args
	}

	// Mount the SSH agent socket
	args = append(args, "-v", fmt.Sprintf("%s:/ssh-agent", sshAuthSock))
	args = append(args, "-e", "SSH_AUTH_SOCK=/ssh-agent")

	// Mount safe SSH files only (config, known_hosts, public keys)
	args = append(args, p.mountSafeSSHFiles(homeDir, username)...)

	return args
}
