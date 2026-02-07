package podman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleSSHForwarding_Disabled(t *testing.T) {
	p := &PodmanProvider{}

	args := p.HandleSSHForwarding(false, "", "/home/test", "testuser", nil)
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(false) returned %v, want empty", args)
	}

	args = p.HandleSSHForwarding(false, "agent", "/home/test", "testuser", nil)
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(false, agent) returned %v, want empty", args)
	}
}

func TestHandleSSHForwarding_DisabledReturnsEmpty(t *testing.T) {
	p := &PodmanProvider{}

	// When forwardKeys is false, any mode should return empty
	modes := []string{"agent", "keys", "proxy", ""}
	for _, mode := range modes {
		args := p.HandleSSHForwarding(false, mode, "/home/test", "testuser", nil)
		if len(args) != 0 {
			t.Errorf("HandleSSHForwarding(false, %q) = %v, want empty", mode, args)
		}
	}
}

func TestHandleSSHForwarding_InvalidMode(t *testing.T) {
	p := &PodmanProvider{}

	args := p.HandleSSHForwarding(true, "invalid", "/home/test", "testuser", nil)
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(true, \"invalid\") = %v, want empty", args)
	}
}

func TestMountSafeSSHFiles(t *testing.T) {
	p := &PodmanProvider{tempDirs: []string{}}

	homeDir := t.TempDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	os.MkdirAll(sshDir, 0700)

	// Create safe and unsafe files
	os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *"), 0644)
	os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("github.com ..."), 0644)
	os.WriteFile(filepath.Join(sshDir, "id_rsa.pub"), []byte("ssh-rsa ..."), 0644)
	os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("ssh-ed25519 ..."), 0644)
	os.WriteFile(filepath.Join(sshDir, "id_rsa"), []byte("PRIVATE KEY"), 0600)
	os.WriteFile(filepath.Join(sshDir, "id_ed25519"), []byte("PRIVATE KEY"), 0600)

	args := p.mountSafeSSHFiles(homeDir, "testuser")
	defer func() {
		for _, dir := range p.tempDirs {
			os.RemoveAll(dir)
		}
	}()

	if len(p.tempDirs) == 0 {
		t.Fatal("Expected temp dir to be created")
	}

	tmpDir := p.tempDirs[0]

	// Check safe files were copied
	for _, name := range []string{"config", "known_hosts", "id_rsa.pub", "id_ed25519.pub"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); os.IsNotExist(err) {
			t.Errorf("Expected safe file %s to be copied", name)
		}
	}

	// Check private keys were NOT copied
	for _, name := range []string{"id_rsa", "id_ed25519"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err == nil {
			t.Errorf("Private key %s should NOT be copied", name)
		}
	}

	// Verify mount args contain the temp dir
	foundMount := false
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], tmpDir) && strings.Contains(args[i+1], ":ro") {
				foundMount = true
				break
			}
		}
	}
	if !foundMount {
		t.Errorf("Expected temp dir mount in args, got: %v", args)
	}
}

func TestMountSafeSSHFiles_NoSSHDir(t *testing.T) {
	p := &PodmanProvider{tempDirs: []string{}}

	homeDir := t.TempDir()
	// No .ssh dir

	args := p.mountSafeSSHFiles(homeDir, "testuser")

	if len(args) != 0 {
		t.Errorf("mountSafeSSHFiles without .ssh returned %v, want empty", args)
	}
	if len(p.tempDirs) != 0 {
		t.Errorf("No temp dirs should be created when .ssh doesn't exist")
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
