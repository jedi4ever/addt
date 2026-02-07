//go:build integration

package podman

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/config"
)

func TestSSHForwarding_Integration_KeysMode(t *testing.T) {
	checkPodmanForSSH(t)

	tmpHome, err := os.MkdirTemp("", "ssh-test-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	testFiles := map[string]string{
		"config":      "Host test\n  Hostname test.example.com\n",
		"known_hosts": "github.com ssh-rsa AAAAB...\n",
		"id_rsa.pub":  "ssh-rsa AAAAB... test@example.com\n",
		"id_rsa":      "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----\n",
	}

	for name, content := range testFiles {
		path := filepath.Join(sshDir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "keys", tmpHome, "testuser", nil)

	foundSSHMount := false
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], ".ssh:ro") {
				foundSSHMount = true
				break
			}
		}
	}

	if !foundSSHMount {
		t.Errorf("Expected SSH directory mount in args, got: %v", args)
	}
}

func TestSSHForwarding_Integration_KeysModeInContainer(t *testing.T) {
	checkPodmanForSSH(t)

	tmpHome, err := os.MkdirTemp("", "ssh-test-container-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	configContent := "Host testhost\n  Hostname test.example.com\n  User testuser\n"
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	podmanPath := config.GetPodmanPath()
	cmd := exec.Command(podmanPath, "run", "--rm",
		"-v", sshDir+":/home/testuser/.ssh:ro",
		"alpine:latest",
		"cat", "/home/testuser/.ssh/config")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "testhost") {
		t.Errorf("Expected SSH config content, got: %s", string(output))
	}
}
