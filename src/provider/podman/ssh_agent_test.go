package podman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	args := p.HandleSSHForwarding(true, "agent", "/home/test", "testuser", nil)

	// Should return empty when no SSH agent
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(true, \"agent\") without SSH_AUTH_SOCK returned %v, want empty", args)
	}
}

func TestHandleSSHForwarding_Agent_MacOSSocket(t *testing.T) {
	p := &PodmanProvider{}

	origSock := os.Getenv("SSH_AUTH_SOCK")
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	// Set a macOS-style socket path (can't be mounted into podman containers)
	os.Setenv("SSH_AUTH_SOCK", "/var/folders/xx/fake/com.apple.launchd.xxx/Listeners")

	args := p.HandleSSHForwarding(true, "agent", "/home/test", "testuser", nil)

	// Should return empty for macOS sockets
	if len(args) != 0 {
		t.Errorf("HandleSSHForwarding(true, \"agent\") with macOS socket returned %v, want empty", args)
	}
}

func TestHandleSSHForwarding_Agent_ValidSocket(t *testing.T) {
	p := &PodmanProvider{}

	// Create a socket path that won't trigger macOS detection
	tmpDir, err := os.MkdirTemp("/tmp", "ssh-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	socketPath := filepath.Join(tmpDir, "agent.sock")

	// Create the socket file (just a regular file for testing)
	if err := os.WriteFile(socketPath, []byte{}, 0600); err != nil {
		t.Fatalf("Failed to create mock socket: %v", err)
	}

	// Create a home directory with .ssh
	homeDir, err := os.MkdirTemp("/tmp", "home-test-*")
	if err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}
	defer os.RemoveAll(homeDir)

	sshDir := filepath.Join(homeDir, ".ssh")
	os.MkdirAll(sshDir, 0700)
	os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *"), 0644)
	os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("github.com ..."), 0644)
	os.WriteFile(filepath.Join(sshDir, "id_rsa.pub"), []byte("ssh-rsa ..."), 0644)

	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Setenv("SSH_AUTH_SOCK", socketPath)
	defer func() {
		if origSock != "" {
			os.Setenv("SSH_AUTH_SOCK", origSock)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	args := p.HandleSSHForwarding(true, "agent", homeDir, "testuser", nil)

	// Should mount the socket
	expectedSocketMount := socketPath + ":/ssh-agent"
	if !containsVolume(args, expectedSocketMount) {
		t.Errorf("HandleSSHForwarding(\"agent\") missing socket mount %q, got %v", expectedSocketMount, args)
	}

	// Should set SSH_AUTH_SOCK env
	if !containsEnv(args, "SSH_AUTH_SOCK=/ssh-agent") {
		t.Errorf("HandleSSHForwarding(\"agent\") missing SSH_AUTH_SOCK env, got %v", args)
	}

	// Should mount safe SSH files (in a temp dir)
	hasSshMount := false
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], ".ssh:ro") {
				hasSshMount = true
				break
			}
		}
	}
	if !hasSshMount {
		t.Errorf("HandleSSHForwarding(\"agent\") missing .ssh mount, got %v", args)
	}
}
