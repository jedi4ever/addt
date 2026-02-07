package podman

import (
	"os"
	"testing"
)

func TestHandleSSHForwarding_Proxy_NoSocket(t *testing.T) {
	p := &PodmanProvider{}

	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		}
	}()

	args := p.HandleSSHForwarding(true, "proxy", "/home/test", "testuser", nil)

	// Should return empty when no SSH agent
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(true, \"proxy\") without SSH_AUTH_SOCK returned %v, want empty", args)
	}
}

func TestHandleSSHForwarding_AllowedKeys_ForcesProxy(t *testing.T) {
	p := &PodmanProvider{}

	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		}
	}()

	// Even in agent mode, allowed keys should force proxy mode
	// Without SSH_AUTH_SOCK, proxy returns empty
	args := p.HandleSSHForwarding(true, "agent", "/home/test", "testuser", []string{"github"})
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding with allowedKeys but no SSH_AUTH_SOCK returned %v, want empty", args)
	}
}
