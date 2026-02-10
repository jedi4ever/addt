//go:build integration

package orbstack

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
	if testing.Short() {
		t.Skip("skipping container test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := provider.DockerCmd("orbstack", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

// createTestProvider creates a minimal OrbStackProvider for testing
func createTestProvider(t *testing.T) *OrbStackProvider {
	t.Helper()
	return &OrbStackProvider{
		tempDirs: []string{},
	}
}

func TestSSHForwarding_Integration_NoForwarding(t *testing.T) {
	checkDockerForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(false, "", "/home/test/.ssh", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for no forwarding, got: %v", args)
	}
}

func TestSSHForwarding_Integration_InvalidMode(t *testing.T) {
	checkDockerForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "invalid", "/home/test/.ssh", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for invalid mode, got: %v", args)
	}
}

func TestSSHForwarding_Integration_NonExistentSSHDir(t *testing.T) {
	checkDockerForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "keys", "/nonexistent/path/.ssh", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for non-existent .ssh dir, got: %v", args)
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

	args := prov.mountSafeSSHFiles(sshDir, "testuser")

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

func TestSSHForwarding_Integration_SafeFilesInContainer(t *testing.T) {
	checkDockerForSSH(t)

	// Create temp home with safe and unsafe SSH files
	tmpHome, err := os.MkdirTemp("", "ssh-container-safe-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	os.MkdirAll(sshDir, 0700)
	os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host test\n  Hostname test.example.com\n"), 0600)
	os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte("github.com ssh-rsa AAAAB...\n"), 0644)
	os.WriteFile(filepath.Join(sshDir, "id_rsa.pub"), []byte("ssh-rsa AAAAB... test@test\n"), 0644)
	os.WriteFile(filepath.Join(sshDir, "id_rsa"), []byte("-----PRIVATE KEY-----\n"), 0600)

	prov := createTestProvider(t)
	defer func() {
		for _, dir := range prov.tempDirs {
			os.RemoveAll(dir)
		}
	}()

	args := prov.mountSafeSSHFiles(sshDir, "testuser")

	if len(prov.tempDirs) == 0 {
		t.Fatal("Expected temp dir to be created")
	}

	// Build docker args to mount the safe dir
	mountArg := ""
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			mountArg = args[i+1]
			break
		}
	}
	if mountArg == "" {
		t.Fatal("No mount argument found")
	}

	// Run container and verify only safe files exist
	cmd := provider.DockerCmd("orbstack", "run", "--rm",
		"-v", mountArg,
		"alpine:latest",
		"ls", "-1", "/home/testuser/.ssh/")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	files := string(output)
	// Safe files should be present
	for _, expected := range []string{"config", "known_hosts", "id_rsa.pub"} {
		if !strings.Contains(files, expected) {
			t.Errorf("Expected %s in container .ssh dir, got: %s", expected, files)
		}
	}
	// Private key should NOT be present
	if strings.Contains(files, "id_rsa\n") {
		t.Errorf("Private key id_rsa should NOT be in container .ssh dir, got: %s", files)
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
		Extensions:     "claude",
		SSHForwardKeys: true,
		SSHForwardMode: "keys",
		NodeVersion:    "22",
		GoVersion:      "1.23.5",
		UvVersion:      "0.4.17",
	}

	prov := &OrbStackProvider{
		config:   cfg,
		tempDirs: []string{},
	}

	args := prov.HandleSSHForwarding(cfg.SSHForwardKeys, cfg.SSHForwardMode, sshDir, "addt", nil)

	if len(args) == 0 {
		t.Error("Expected SSH mount args")
	}

	t.Logf("SSH forwarding args: %v", args)
}
