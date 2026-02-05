package podman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHandleSSHForwarding_Disabled(t *testing.T) {
	p := &PodmanProvider{}

	testCases := []string{"", "off", "false", "none"}

	for _, mode := range testCases {
		t.Run(mode, func(t *testing.T) {
			args := p.HandleSSHForwarding(mode, "/home/test", "testuser")
			if len(args) != 0 {
				t.Errorf("HandleSSHForwarding(%q) returned %v, want empty", mode, args)
			}
		})
	}
}

func TestHandleSSHForwarding_Keys(t *testing.T) {
	p := &PodmanProvider{}

	// Create a temporary home directory with .ssh
	homeDir := t.TempDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	// Create some test files
	os.WriteFile(filepath.Join(sshDir, "id_rsa"), []byte("private"), 0600)
	os.WriteFile(filepath.Join(sshDir, "id_rsa.pub"), []byte("public"), 0644)
	os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *"), 0644)

	args := p.HandleSSHForwarding("keys", homeDir, "testuser")

	// Should mount .ssh directory
	expectedMount := sshDir + ":/home/testuser/.ssh:ro"
	if !containsVolume(args, expectedMount) {
		t.Errorf("HandleSSHForwarding(\"keys\") missing mount %q, got %v", expectedMount, args)
	}

	// Should NOT set SSH_AUTH_SOCK
	if containsEnvPrefix(args, "SSH_AUTH_SOCK=") {
		t.Errorf("HandleSSHForwarding(\"keys\") should not set SSH_AUTH_SOCK")
	}
}

func TestHandleSSHForwarding_Keys_NoSSHDir(t *testing.T) {
	p := &PodmanProvider{}

	// Create a temporary home directory WITHOUT .ssh
	homeDir := t.TempDir()

	args := p.HandleSSHForwarding("keys", homeDir, "testuser")

	// Should return empty when .ssh doesn't exist
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(\"keys\") without .ssh returned %v, want empty", args)
	}
}

func TestHandleSSHForwarding_Agent_NoSocket(t *testing.T) {
	p := &PodmanProvider{}

	// Save and clear SSH_AUTH_SOCK
	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		}
	}()

	args := p.HandleSSHForwarding("agent", "/home/test", "testuser")

	// Should return empty when no SSH agent
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(\"agent\") without SSH_AUTH_SOCK returned %v, want empty", args)
	}
}

// Helper functions

func containsVolume(args []string, mount string) bool {
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) && args[i+1] == mount {
			return true
		}
	}
	return false
}

func containsEnvPrefix(args []string, prefix string) bool {
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && len(args[i+1]) >= len(prefix) && args[i+1][:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func containsEnv(args []string, env string) bool {
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && args[i+1] == env {
			return true
		}
	}
	return false
}
