package podman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/util"
)

// HandleSSHForwarding configures SSH forwarding based on config.
// When forwardKeys is true, the forwardMode determines the method:
//   - "proxy": Forward filtered SSH agent (only allowed keys)
//   - "agent": Forward SSH agent socket
//   - "keys": Mount ~/.ssh directory read-only
//
// If allowedKeys is set, proxy mode is automatically enabled for agent forwarding
func (p *PodmanProvider) HandleSSHForwarding(forwardKeys bool, forwardMode, homeDir, username string, allowedKeys []string) []string {
	if !forwardKeys {
		return nil
	}

	// If allowed keys are specified, use proxy mode regardless of forwardMode setting
	if len(allowedKeys) > 0 && (forwardMode == "agent" || forwardMode == "proxy") {
		return p.handleSSHProxyForwarding(homeDir, username, allowedKeys)
	}

	if forwardMode == "proxy" {
		// Proxy mode without filters - just forward all keys through proxy
		return p.handleSSHProxyForwarding(homeDir, username, nil)
	} else if forwardMode == "agent" {
		return p.handleSSHAgentForwarding(homeDir, username)
	} else if forwardMode == "keys" {
		return p.handleSSHKeysForwarding(homeDir, username)
	}

	return nil
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

	// Set restrictive permissions and write PID file
	if err := os.Chmod(tmpDir, 0700); err != nil {
		os.RemoveAll(tmpDir)
		return args
	}
	if err := security.WritePIDFile(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
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
