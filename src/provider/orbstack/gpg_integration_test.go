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

// checkDockerForGPG verifies Docker is available
func checkDockerForGPG(t *testing.T) {
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

func TestGPGForwarding_Integration_Enabled(t *testing.T) {
	checkDockerForGPG(t)

	// Create a temp home directory with .gnupg
	tmpHome, err := os.MkdirTemp("", "gpg-test-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	gnupgDir := filepath.Join(tmpHome, ".gnupg")
	if err := os.MkdirAll(gnupgDir, 0700); err != nil {
		t.Fatalf("Failed to create .gnupg dir: %v", err)
	}

	// Create some test GPG files
	testFiles := map[string]string{
		"pubring.kbx":    "mock pubring data",
		"trustdb.gpg":    "mock trustdb data",
		"gpg.conf":       "# GPG config\nkeyserver hkps://keys.openpgp.org\n",
		"gpg-agent.conf": "# GPG agent config\n",
	}

	for name, content := range testFiles {
		path := filepath.Join(gnupgDir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	prov := &OrbStackProvider{tempDirs: []string{}}
	args := prov.HandleGPGForwarding("keys", gnupgDir, "testuser", nil)

	// Should have volume mount for .gnupg
	foundGPGMount := false
	for i, arg := range args {
		if arg == "-v" && i+1 < len(args) {
			if strings.Contains(args[i+1], ".gnupg") {
				foundGPGMount = true
				break
			}
		}
	}

	if !foundGPGMount {
		t.Errorf("Expected GPG directory mount in args, got: %v", args)
	}

	// Should have GPG_TTY env var
	foundGPGTTY := false
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) {
			if args[i+1] == "GPG_TTY=/dev/console" {
				foundGPGTTY = true
				break
			}
		}
	}

	if !foundGPGTTY {
		t.Errorf("Expected GPG_TTY env var in args, got: %v", args)
	}
}

func TestGPGForwarding_Integration_MountInContainer(t *testing.T) {
	checkDockerForGPG(t)

	// Create a temp home directory with .gnupg
	tmpHome, err := os.MkdirTemp("", "gpg-test-container-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	gnupgDir := filepath.Join(tmpHome, ".gnupg")
	if err := os.MkdirAll(gnupgDir, 0700); err != nil {
		t.Fatalf("Failed to create .gnupg dir: %v", err)
	}

	// Create test config
	configContent := "# GPG test config\nkeyserver hkps://keys.openpgp.org\n"
	if err := os.WriteFile(filepath.Join(gnupgDir, "gpg.conf"), []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create gpg.conf: %v", err)
	}

	// Run container with GPG mount and verify files are accessible
	cmd := provider.DockerCmd("orbstack", "run", "--rm",
		"-v", gnupgDir+":/home/testuser/.gnupg",
		"-e", "GPG_TTY=/dev/console",
		"alpine:latest",
		"cat", "/home/testuser/.gnupg/gpg.conf")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "keyserver") {
		t.Errorf("Expected GPG config content, got: %s", string(output))
	}
}

func TestGPGForwarding_Integration_Disabled(t *testing.T) {
	checkDockerForGPG(t)

	prov := &OrbStackProvider{tempDirs: []string{}}
	args := prov.HandleGPGForwarding("", "/home/test/.gnupg", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for disabled GPG forwarding, got: %v", args)
	}
}

func TestGPGForwarding_Integration_NonExistentGnupgDir(t *testing.T) {
	checkDockerForGPG(t)

	prov := &OrbStackProvider{tempDirs: []string{}}
	args := prov.HandleGPGForwarding("keys", "/nonexistent/path/.gnupg", "testuser", nil)

	if len(args) != 0 {
		t.Errorf("Expected empty args for non-existent .gnupg dir, got: %v", args)
	}
}

func TestGPGForwarding_Integration_FullProviderWithGPG(t *testing.T) {
	checkDockerForGPG(t)

	// Create temp .gnupg dir
	tmpHome, err := os.MkdirTemp("", "gpg-provider-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	gnupgDir := filepath.Join(tmpHome, ".gnupg")
	if err := os.MkdirAll(gnupgDir, 0700); err != nil {
		t.Fatalf("Failed to create .gnupg dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(gnupgDir, "gpg.conf"), []byte("# GPG config"), 0600); err != nil {
		t.Fatalf("Failed to create gpg.conf: %v", err)
	}

	// Create a full provider config
	cfg := &provider.Config{
		Extensions:  "claude",
		GPGForward:  "keys",
		NodeVersion: "22",
		GoVersion:   "1.23.5",
		UvVersion:   "0.4.17",
	}

	prov := &OrbStackProvider{
		config:   cfg,
		tempDirs: []string{},
	}

	args := prov.HandleGPGForwarding(cfg.GPGForward, gnupgDir, "addt", cfg.GPGAllowedKeyIDs)

	if len(args) == 0 {
		t.Error("Expected GPG mount args")
	}

	t.Logf("GPG forwarding args: %v", args)
}

func TestGPGForwarding_Integration_VerifyGPGTTYInContainer(t *testing.T) {
	checkDockerForGPG(t)

	// Run container and verify GPG_TTY is set correctly
	cmd := provider.DockerCmd("orbstack", "run", "--rm",
		"-e", "GPG_TTY=/dev/console",
		"alpine:latest",
		"sh", "-c", "echo $GPG_TTY")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "/dev/console") {
		t.Errorf("Expected GPG_TTY=/dev/console, got: %s", string(output))
	}
}
