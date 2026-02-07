//go:build integration

package docker

import (
	"os"
	"testing"
)

func TestSSHForwarding_Integration_AgentModeNoSocket(t *testing.T) {
	checkDockerForSSH(t)

	// Unset SSH_AUTH_SOCK temporarily
	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		}
	}()

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "agent", "/home/test", "testuser", nil)

	// Should return empty args when no socket
	if len(args) > 0 {
		t.Errorf("Expected empty args when SSH_AUTH_SOCK not set, got: %v", args)
	}
}
