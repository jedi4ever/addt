package docker

import (
	"fmt"
	"os"
	"path/filepath"
)

// HandleGPGForwarding configures GPG forwarding by mounting ~/.gnupg directory.
// When enabled, mounts the user's GPG directory and sets GPG_TTY for signing operations.
func (p *DockerProvider) HandleGPGForwarding(gpgForward bool, homeDir, username string) []string {
	var args []string

	if !gpgForward {
		return args
	}

	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if _, err := os.Stat(gnupgDir); err != nil {
		return args
	}

	// Mount GPG directory
	args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg", gnupgDir, username))

	// Set GPG_TTY for interactive signing
	args = append(args, "-e", "GPG_TTY=/dev/console")

	return args
}
