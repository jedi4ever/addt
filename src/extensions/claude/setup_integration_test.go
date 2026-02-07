//go:build integration

package claude

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/provider/docker"
)

const testImageName = "addt-test-claude-setup"

// checkDocker verifies Docker is available
func checkDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

// createTestProvider creates a provider for tests
func createTestProvider(t *testing.T, cfg *provider.Config) provider.Provider {
	t.Helper()
	prov, err := docker.NewDockerProvider(
		cfg,
		assets.DockerDockerfile,
		assets.DockerDockerfileBase,
		assets.DockerEntrypoint,
		assets.DockerInitFirewall,
		assets.DockerInstallSh,
		extensions.FS,
	)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}
	return prov
}

// ensureTestImage builds the test image if needed
func ensureTestImage(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping image build test in short mode")
	}

	cmd := exec.Command("docker", "image", "inspect", testImageName)
	if cmd.Run() == nil {
		return // Image exists
	}

	cfg := &provider.Config{
		AddtVersion: "0.0.0-test",
		Extensions:  "claude",
		NodeVersion: "22",
		GoVersion:   "latest",
		UvVersion:   "latest",
		ImageName:   testImageName,
	}

	prov := createTestProvider(t, cfg)
	if err := prov.Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("Failed to build test image: %v", err)
	}
}

// TestSetup_Integration_PreservesExistingAuth verifies that setup.sh preserves
// existing OAuth credentials when the user has already completed onboarding
func TestSetup_Integration_PreservesExistingAuth(t *testing.T) {
	checkDocker(t)
	ensureTestImage(t)

	// Run container with existing config that has hasCompletedOnboarding
	// The setup.sh should detect this and NOT overwrite with API key config
	cmd := exec.Command("docker", "run", "--rm",
		"-e", "ANTHROPIC_API_KEY=sk-ant-test-key-12345678901234567890",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Create existing config with OAuth-like setup
			cat > ~/.claude.json << 'EXISTING'
{
  "hasCompletedOnboarding": true,
  "hasTrustDialogAccepted": true,
  "oauthAccount": "user@example.com"
}
EXISTING
			# Run setup.sh
			/usr/local/share/addt/extensions/claude/setup.sh
			# Output the resulting config
			echo "=== CONFIG ==="
			cat ~/.claude.json
		`,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outputStr)
	}

	// Should detect existing config and not modify it
	if !strings.Contains(outputStr, "not modifying") {
		t.Errorf("Expected setup.sh to detect existing config and not modify.\nOutput: %s", outputStr)
	}

	// Should preserve oauthAccount
	if !strings.Contains(outputStr, "oauthAccount") {
		t.Errorf("Expected OAuth credentials to be preserved.\nOutput: %s", outputStr)
	}

	// Should NOT have customApiKeyResponses (that would mean it was overwritten)
	if strings.Contains(outputStr, "customApiKeyResponses") {
		t.Errorf("Config was overwritten with API key setup.\nOutput: %s", outputStr)
	}

	t.Logf("Output: %s", outputStr)
}

// TestSetup_Integration_CreatesConfigWithAPIKey verifies that setup.sh creates
// proper config when no existing config exists and ANTHROPIC_API_KEY is set
func TestSetup_Integration_CreatesConfigWithAPIKey(t *testing.T) {
	checkDocker(t)
	ensureTestImage(t)

	// Run container without existing config but with API key
	cmd := exec.Command("docker", "run", "--rm",
		"-e", "ANTHROPIC_API_KEY=sk-ant-test-key-12345678901234567890",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Remove any existing config
			rm -f ~/.claude.json
			# Run setup.sh
			/usr/local/share/addt/extensions/claude/setup.sh
			# Output the resulting config
			echo "=== CONFIG ==="
			cat ~/.claude.json
			echo "=== INTERNAL ==="
			cat ~/.claude/claude.json 2>/dev/null || echo "no internal config"
		`,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outputStr)
	}

	// Should have hasCompletedOnboarding
	if !strings.Contains(outputStr, `"hasCompletedOnboarding": true`) {
		t.Errorf("Expected hasCompletedOnboarding in config.\nOutput: %s", outputStr)
	}

	// Should have customApiKeyResponses
	if !strings.Contains(outputStr, "customApiKeyResponses") {
		t.Errorf("Expected customApiKeyResponses in config.\nOutput: %s", outputStr)
	}

	// Should have last 20 chars of API key
	if !strings.Contains(outputStr, "12345678901234567890") {
		t.Errorf("Expected last 20 chars of API key.\nOutput: %s", outputStr)
	}

	// Should have /workspace trust
	if !strings.Contains(outputStr, "/workspace") {
		t.Errorf("Expected /workspace trust in config.\nOutput: %s", outputStr)
	}

	// Should have internal config with hasTrustDialogHooksAccepted
	if !strings.Contains(outputStr, "hasTrustDialogHooksAccepted") {
		t.Errorf("Expected hasTrustDialogHooksAccepted in internal config.\nOutput: %s", outputStr)
	}

	t.Logf("Output: %s", outputStr)
}

// TestSetup_Integration_NoAPIKeyNoConfig verifies that setup.sh does nothing
// when no API key is set and no existing config exists
func TestSetup_Integration_NoAPIKeyNoConfig(t *testing.T) {
	checkDocker(t)
	ensureTestImage(t)

	// Run container without API key and without existing config
	cmd := exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Remove any existing config
			rm -f ~/.claude.json
			# Unset API key
			unset ANTHROPIC_API_KEY
			# Run setup.sh
			/usr/local/share/addt/extensions/claude/setup.sh
			# Check if config was created
			if [ -f ~/.claude.json ]; then
				echo "CONFIG_CREATED"
				cat ~/.claude.json
			else
				echo "NO_CONFIG"
			fi
		`,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, outputStr)
	}

	// Config should NOT be created
	if strings.Contains(outputStr, "CONFIG_CREATED") {
		t.Errorf("Expected no config to be created without API key.\nOutput: %s", outputStr)
	}

	if !strings.Contains(outputStr, "NO_CONFIG") {
		t.Errorf("Expected NO_CONFIG marker.\nOutput: %s", outputStr)
	}

	t.Logf("Output: %s", outputStr)
}
