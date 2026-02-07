//go:build integration

package podman

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/provider"
)

// checkPodmanForSSH verifies Podman is available
func checkPodmanForSSH(t *testing.T) {
	t.Helper()
	podmanPath := config.GetPodmanPath()
	if podmanPath == "" {
		t.Skip("Podman not found, skipping integration test")
	}
	cmd := exec.Command(podmanPath, "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Podman not running, skipping integration test")
	}
}

// createTestProvider creates a minimal PodmanProvider for testing
func createTestProvider(t *testing.T) *PodmanProvider {
	t.Helper()
	return &PodmanProvider{
		tempDirs: []string{},
	}
}

func TestSSHForwarding_Integration_NoForwarding(t *testing.T) {
	checkPodmanForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(false, "", "/home/test", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for no forwarding, got: %v", args)
	}
}

func TestSSHForwarding_Integration_InvalidMode(t *testing.T) {
	checkPodmanForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "invalid", "/home/test", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for invalid mode, got: %v", args)
	}
}

func TestSSHForwarding_Integration_NonExistentSSHDir(t *testing.T) {
	checkPodmanForSSH(t)

	prov := createTestProvider(t)
	args := prov.HandleSSHForwarding(true, "keys", "/nonexistent/path", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for non-existent .ssh dir, got: %v", args)
	}
}

func TestSSHForwarding_Integration_MountSafeFiles(t *testing.T) {
	checkPodmanForSSH(t)

	tmpHome, err := os.MkdirTemp("", "ssh-safe-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	safeFiles := []string{"config", "known_hosts", "id_rsa.pub", "id_ed25519.pub"}
	unsafeFiles := []string{"id_rsa", "id_ed25519"}

	for _, name := range safeFiles {
		if err := os.WriteFile(filepath.Join(sshDir, name), []byte("safe content"), 0600); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	for _, name := range unsafeFiles {
		if err := os.WriteFile(filepath.Join(sshDir, name), []byte("private key content"), 0600); err != nil {
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

	if len(prov.tempDirs) == 0 {
		t.Fatal("Expected temp dir to be created")
	}

	tmpDir := prov.tempDirs[0]

	// Check safe files were copied
	for _, name := range []string{"config", "known_hosts", "id_rsa.pub"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); os.IsNotExist(err) {
			t.Errorf("Expected safe file %s to be copied", name)
		}
	}

	// Check private keys were NOT copied
	for _, name := range unsafeFiles {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err == nil {
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
	checkPodmanForSSH(t)

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

	args := prov.mountSafeSSHFiles(tmpHome, "testuser")

	if len(prov.tempDirs) == 0 {
		t.Fatal("Expected temp dir to be created")
	}

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
	podmanPath := config.GetPodmanPath()
	cmd := exec.Command(podmanPath, "run", "--rm",
		"-v", mountArg,
		"alpine:latest",
		"ls", "-1", "/home/testuser/.ssh/")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	files := string(output)
	for _, expected := range []string{"config", "known_hosts", "id_rsa.pub"} {
		if !strings.Contains(files, expected) {
			t.Errorf("Expected %s in container .ssh dir, got: %s", expected, files)
		}
	}
	if strings.Contains(files, "id_rsa\n") {
		t.Errorf("Private key id_rsa should NOT be in container .ssh dir, got: %s", files)
	}
}

func TestSSHForwarding_Integration_FullProviderWithSSH(t *testing.T) {
	checkPodmanForSSH(t)

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

	cfg := &provider.Config{
		Extensions:     "claude",
		SSHForwardKeys: true,
		SSHForwardMode: "keys",
		NodeVersion:    "22",
		GoVersion:      "1.23.5",
		UvVersion:      "0.4.17",
	}

	prov := &PodmanProvider{
		config:   cfg,
		tempDirs: []string{},
	}

	args := prov.HandleSSHForwarding(cfg.SSHForwardKeys, cfg.SSHForwardMode, tmpHome, "addt", nil)

	if len(args) == 0 {
		t.Error("Expected SSH mount args")
	}

	t.Logf("SSH forwarding args: %v", args)
}
