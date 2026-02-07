//go:build integration

package podman

import (
	"os"
	"testing"
)

func TestSSHForwarding_Integration_ProxyModeNoSocket(t *testing.T) {
	checkPodmanForSSH(t)

	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		}
	}()

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "proxy", "/home/test", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for proxy mode without SSH_AUTH_SOCK, got: %v", args)
	}
}

func TestSSHForwarding_Integration_AllowedKeysNoSocket(t *testing.T) {
	checkPodmanForSSH(t)

	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		}
	}()

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "agent", "/home/test", "testuser", []string{"github"})

	if len(args) != 0 {
		t.Errorf("Expected empty args for allowed keys without SSH_AUTH_SOCK, got: %v", args)
	}
}
