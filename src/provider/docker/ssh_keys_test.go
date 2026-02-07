package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHandleSSHForwarding_Keys(t *testing.T) {
	p := &DockerProvider{}

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

	args := p.HandleSSHForwarding(true, "keys", homeDir, "testuser", nil)

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
	p := &DockerProvider{}

	// Create a temporary home directory WITHOUT .ssh
	homeDir := t.TempDir()

	args := p.HandleSSHForwarding(true, "keys", homeDir, "testuser", nil)

	// Should return empty when .ssh doesn't exist
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(\"keys\") without .ssh returned %v, want empty", args)
	}
}

func TestHandleSSHForwarding_AllowedKeys_IgnoredForKeysMode(t *testing.T) {
	p := &DockerProvider{}

	// Create a temporary home directory with .ssh
	homeDir := t.TempDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	os.MkdirAll(sshDir, 0700)
	os.WriteFile(filepath.Join(sshDir, "id_rsa"), []byte("private"), 0600)

	// In keys mode, allowedKeys should be ignored (keys mode mounts full .ssh)
	args := p.HandleSSHForwarding(true, "keys", homeDir, "testuser", []string{"github"})

	expectedMount := sshDir + ":/home/testuser/.ssh:ro"
	if !containsVolume(args, expectedMount) {
		t.Errorf("HandleSSHForwarding(\"keys\") with allowedKeys should still mount .ssh, got %v", args)
	}
}
