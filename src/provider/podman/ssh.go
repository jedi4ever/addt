package podman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/util"
)

// HandleSSHForwarding configures SSH forwarding based on config.
// Modes:
//   - "agent" or "true": Forward SSH agent socket (not supported on macOS)
//   - "keys": Mount ~/.ssh directory read-only
//   - "" or other: No SSH forwarding
func (p *PodmanProvider) HandleSSHForwarding(sshForward, homeDir, username string) []string {
	var args []string

	if sshForward == "agent" || sshForward == "true" {
		args = p.handleSSHAgentForwarding(homeDir, username)
	} else if sshForward == "keys" {
		args = p.handleSSHKeysForwarding(homeDir, username)
	}

	return args
}

// handleSSHAgentForwarding forwards the SSH agent socket into the container
func (p *PodmanProvider) handleSSHAgentForwarding(homeDir, username string) []string {
	var args []string

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return args
	}

	// Check if socket exists and is accessible
	if _, err := os.Stat(sshAuthSock); err != nil {
		return args
	}

	// Check for macOS launchd sockets (won't work in Podman)
	if strings.Contains(sshAuthSock, "com.apple.launchd") || strings.Contains(sshAuthSock, "/var/folders/") {
		fmt.Println("Warning: SSH agent forwarding not supported on macOS (use ADDT_SSH_FORWARD=keys)")
		return args
	}

	// Mount the SSH agent socket
	args = append(args, "-v", fmt.Sprintf("%s:/ssh-agent", sshAuthSock))
	args = append(args, "-e", "SSH_AUTH_SOCK=/ssh-agent")

	// Mount safe SSH files only (config, known_hosts, public keys)
	args = append(args, p.mountSafeSSHFiles(homeDir, username)...)

	return args
}

// handleSSHKeysForwarding mounts the entire ~/.ssh directory read-only
func (p *PodmanProvider) handleSSHKeysForwarding(homeDir, username string) []string {
	var args []string

	sshDir := filepath.Join(homeDir, ".ssh")
	if _, err := os.Stat(sshDir); err == nil {
		args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.ssh:ro", sshDir, username))
	}

	return args
}

// mountSafeSSHFiles creates a temp directory with only safe SSH files
// (config, known_hosts, public keys) and returns mount arguments
func (p *PodmanProvider) mountSafeSSHFiles(homeDir, username string) []string {
	var args []string

	sshDir := filepath.Join(homeDir, ".ssh")
	if _, err := os.Stat(sshDir); err != nil {
		return args
	}

	tmpDir, err := os.MkdirTemp("", "ssh-safe-*")
	if err != nil {
		return args
	}

	p.tempDirs = append(p.tempDirs, tmpDir)

	// Copy safe files only
	util.SafeCopyFile(filepath.Join(sshDir, "config"), filepath.Join(tmpDir, "config"))
	util.SafeCopyFile(filepath.Join(sshDir, "known_hosts"), filepath.Join(tmpDir, "known_hosts"))

	// Copy public keys
	files, _ := filepath.Glob(filepath.Join(sshDir, "*.pub"))
	for _, f := range files {
		util.SafeCopyFile(f, filepath.Join(tmpDir, filepath.Base(f)))
	}

	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.ssh:ro", tmpDir, username))

	return args
}
