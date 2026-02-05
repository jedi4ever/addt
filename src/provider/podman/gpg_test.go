package podman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHandleGPGForwarding_Disabled(t *testing.T) {
	p := &PodmanProvider{}

	args := p.HandleGPGForwarding(false, "/home/test", "testuser")

	if len(args) != 0 {
		t.Errorf("HandleGPGForwarding(false) returned %v, want empty", args)
	}
}

func TestHandleGPGForwarding_Enabled_NoGnupgDir(t *testing.T) {
	p := &PodmanProvider{}

	// Create a temporary home directory WITHOUT .gnupg
	homeDir := t.TempDir()

	args := p.HandleGPGForwarding(true, homeDir, "testuser")

	// Should return empty when .gnupg doesn't exist
	if len(args) != 0 {
		t.Errorf("HandleGPGForwarding(true) without .gnupg returned %v, want empty", args)
	}
}

func TestHandleGPGForwarding_Enabled_WithGnupgDir(t *testing.T) {
	p := &PodmanProvider{}

	// Create a temporary home directory with .gnupg
	homeDir := t.TempDir()
	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if err := os.MkdirAll(gnupgDir, 0700); err != nil {
		t.Fatalf("Failed to create .gnupg dir: %v", err)
	}

	args := p.HandleGPGForwarding(true, homeDir, "testuser")

	// Should mount .gnupg directory
	expectedMount := gnupgDir + ":/home/testuser/.gnupg"
	if !containsVolume(args, expectedMount) {
		t.Errorf("HandleGPGForwarding(true) missing mount %q, got %v", expectedMount, args)
	}

	// Should set GPG_TTY
	if !containsEnv(args, "GPG_TTY=/dev/console") {
		t.Errorf("HandleGPGForwarding(true) missing GPG_TTY env var, got %v", args)
	}
}
