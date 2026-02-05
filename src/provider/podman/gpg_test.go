package podman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleGPGForwarding_Disabled(t *testing.T) {
	p := &PodmanProvider{tempDirs: []string{}}

	testCases := []string{"", "off", "false", "none"}

	for _, mode := range testCases {
		t.Run(mode, func(t *testing.T) {
			args := p.HandleGPGForwarding(mode, "/home/test", "testuser", nil)
			if len(args) != 0 {
				t.Errorf("HandleGPGForwarding(%q) returned %v, want empty", mode, args)
			}
		})
	}
}

func TestHandleGPGForwarding_Keys(t *testing.T) {
	p := &PodmanProvider{tempDirs: []string{}}

	// Create a temporary home directory with .gnupg
	homeDir := t.TempDir()
	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if err := os.MkdirAll(gnupgDir, 0700); err != nil {
		t.Fatalf("Failed to create .gnupg dir: %v", err)
	}

	// Create some test files
	os.WriteFile(filepath.Join(gnupgDir, "pubring.kbx"), []byte("pubring"), 0600)
	os.WriteFile(filepath.Join(gnupgDir, "trustdb.gpg"), []byte("trustdb"), 0600)

	args := p.HandleGPGForwarding("keys", homeDir, "testuser", nil)

	// Should mount .gnupg directory read-only
	foundMount := false
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], ".gnupg:ro") {
				foundMount = true
				break
			}
		}
	}
	if !foundMount {
		t.Errorf("HandleGPGForwarding(\"keys\") missing read-only mount, got %v", args)
	}

	// Should set GPG_TTY
	if !containsEnv(args, "GPG_TTY=/dev/console") {
		t.Errorf("HandleGPGForwarding(\"keys\") missing GPG_TTY env, got %v", args)
	}
}

func TestHandleGPGForwarding_Keys_NoGnupgDir(t *testing.T) {
	p := &PodmanProvider{tempDirs: []string{}}

	// Create a temporary home directory WITHOUT .gnupg
	homeDir := t.TempDir()

	args := p.HandleGPGForwarding("keys", homeDir, "testuser", nil)

	// Should return empty when .gnupg doesn't exist
	if len(args) != 0 {
		t.Errorf("HandleGPGForwarding(\"keys\") without .gnupg returned %v, want empty", args)
	}
}

func TestHandleGPGForwarding_LegacyTrue(t *testing.T) {
	p := &PodmanProvider{tempDirs: []string{}}

	// Create a temporary home directory with .gnupg
	homeDir := t.TempDir()
	gnupgDir := filepath.Join(homeDir, ".gnupg")
	if err := os.MkdirAll(gnupgDir, 0700); err != nil {
		t.Fatalf("Failed to create .gnupg dir: %v", err)
	}

	os.WriteFile(filepath.Join(gnupgDir, "pubring.kbx"), []byte("pubring"), 0600)

	// "true" should behave like "keys" for backward compatibility
	args := p.HandleGPGForwarding("true", homeDir, "testuser", nil)

	if len(args) == 0 {
		t.Errorf("HandleGPGForwarding(\"true\") returned empty args")
	}

	// Should set GPG_TTY
	if !containsEnv(args, "GPG_TTY=/dev/console") {
		t.Errorf("HandleGPGForwarding(\"true\") missing GPG_TTY env, got %v", args)
	}
}

func TestHandleGPGForwarding_InvalidMode(t *testing.T) {
	p := &PodmanProvider{tempDirs: []string{}}

	args := p.HandleGPGForwarding("invalid", "/home/test", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("HandleGPGForwarding(\"invalid\") returned %v, want empty", args)
	}
}
