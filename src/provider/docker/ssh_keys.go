package docker

import (
	"fmt"
	"os"
	"path/filepath"
)

// handleSSHKeysForwarding mounts the entire ~/.ssh directory read-only
func (p *DockerProvider) handleSSHKeysForwarding(homeDir, username string) []string {
	var args []string

	sshDir := filepath.Join(homeDir, ".ssh")
	if _, err := os.Stat(sshDir); err == nil {
		args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.ssh:ro", sshDir, username))
	}

	return args
}
