//go:build integration

package docker

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/provider"
)

// checkDockerForSSH verifies Docker is available
func checkDockerForSSH(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

// createTestProvider creates a minimal DockerProvider for testing
func createTestProvider(t *testing.T) *DockerProvider {
	t.Helper()
	return &DockerProvider{
		tempDirs: []string{},
	}
}

func TestSSHForwarding_Integration_KeysMode(t *testing.T) {
	checkDockerForSSH(t)

	// Create a temp home directory with .ssh
	tmpHome, err := os.MkdirTemp("", "ssh-test-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	// Create some test SSH files
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
	args := prov.HandleSSHForwarding("keys", tmpHome, "testuser")

	// Should have volume mount for .ssh
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
	checkDockerForSSH(t)

	// Create a temp home directory with .ssh
	tmpHome, err := os.MkdirTemp("", "ssh-test-container-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	// Create test config
	configContent := "Host testhost\n  Hostname test.example.com\n  User testuser\n"
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Run container with SSH mount and verify files are accessible
	cmd := exec.Command("docker", "run", "--rm",
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
	args := prov.HandleSSHForwarding("agent", "/home/test", "testuser")

	// Should return empty args when no socket
	if len(args) > 0 {
		t.Errorf("Expected empty args when SSH_AUTH_SOCK not set, got: %v", args)
	}
}

func TestSSHForwarding_Integration_NoForwarding(t *testing.T) {
	checkDockerForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding("", "/home/test", "testuser")

	if len(args) != 0 {
		t.Errorf("Expected empty args for no forwarding, got: %v", args)
	}
}

func TestSSHForwarding_Integration_InvalidMode(t *testing.T) {
	checkDockerForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding("invalid", "/home/test", "testuser")

	if len(args) != 0 {
		t.Errorf("Expected empty args for invalid mode, got: %v", args)
	}
}

func TestSSHForwarding_Integration_MountSafeFiles(t *testing.T) {
	checkDockerForSSH(t)

	// Create a temp home directory with .ssh containing sensitive and safe files
	tmpHome, err := os.MkdirTemp("", "ssh-safe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	// Create safe and unsafe files
	safeFiles := []string{"config", "known_hosts", "id_rsa.pub", "id_ed25519.pub"}
	unsafeFiles := []string{"id_rsa", "id_ed25519"}

	for _, name := range safeFiles {
		path := filepath.Join(sshDir, name)
		if err := os.WriteFile(path, []byte("safe content"), 0600); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	for _, name := range unsafeFiles {
		path := filepath.Join(sshDir, name)
		if err := os.WriteFile(path, []byte("private key content"), 0600); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	prov := createTestProvider(t)
	defer func() {
		for _, dir := range prov.tempDirs {
			os.RemoveAll(dir)
		}
	}()

	args := prov.mountSafeSSHFiles(tmpHome, "testuser")

	// Should have created a temp dir and mounted it
	if len(prov.tempDirs) == 0 {
		t.Fatal("Expected temp dir to be created")
	}

	tmpDir := prov.tempDirs[0]

	// Check safe files were copied
	for _, name := range []string{"config", "known_hosts"} {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected safe file %s to be copied", name)
		}
	}

	// Check public keys were copied
	pubKeyPath := filepath.Join(tmpDir, "id_rsa.pub")
	if _, err := os.Stat(pubKeyPath); os.IsNotExist(err) {
		t.Error("Expected public key to be copied")
	}

	// Check private keys were NOT copied
	for _, name := range unsafeFiles {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("Private key %s should NOT be copied", name)
		}
	}

	// Verify mount args
	foundMount := false
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], tmpDir) && strings.HasSuffix(args[i+1], ":ro") {
				foundMount = true
				break
			}
		}
	}

	if !foundMount {
		t.Errorf("Expected temp dir mount in args, got: %v", args)
	}
}

func TestSSHForwarding_Integration_NonExistentSSHDir(t *testing.T) {
	checkDockerForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding("keys", "/nonexistent/path", "testuser")

	if len(args) != 0 {
		t.Errorf("Expected empty args for non-existent .ssh dir, got: %v", args)
	}
}

func TestSSHForwarding_Integration_FullProviderWithSSH(t *testing.T) {
	checkDockerForSSH(t)

	// Create temp SSH dir
	tmpHome, err := os.MkdirTemp("", "ssh-provider-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte("# SSH config"), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create a full provider config
	cfg := &provider.Config{
		Extensions:  "claude",
		SSHForward:  "keys",
		NodeVersion: "22",
		GoVersion:   "1.23",
		UvVersion:   "0.4.17",
	}

	prov := &DockerProvider{
		config:   cfg,
		tempDirs: []string{},
	}

	args := prov.HandleSSHForwarding(cfg.SSHForward, tmpHome, "addt")

	if len(args) == 0 {
		t.Error("Expected SSH mount args")
	}

	t.Logf("SSH forwarding args: %v", args)
}
