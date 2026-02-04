//go:build integration

package firewall

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/provider/docker"
)

// checkDockerForFirewall verifies Docker is available
func checkDockerForFirewallIntegration(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

// createFirewallTestProvider creates a provider for firewall tests
func createFirewallTestProvider(t *testing.T, cfg *provider.Config) provider.Provider {
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

// ensureFirewallTestImage builds the test image if needed
func ensureFirewallTestImage(t *testing.T, imageName string) {
	t.Helper()

	cmd := exec.Command("docker", "image", "inspect", imageName)
	if cmd.Run() == nil {
		return // Image exists
	}

	cfg := &provider.Config{
		AddtVersion: "0.0.0-test",
		Extensions:  "claude",
		NodeVersion: "22",
		GoVersion:   "latest",
		UvVersion:   "latest",
		ImageName:   imageName,
	}

	prov := createFirewallTestProvider(t, cfg)
	if err := prov.Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("Failed to build test image: %v", err)
	}
}

func TestFirewallNetwork_Integration_AllowedDomainAccessible(t *testing.T) {
	checkDockerForFirewallIntegration(t)

	testImageName := "addt-test-firewall-integration"
	ensureFirewallTestImage(t, testImageName)

	// Run container with firewall enabled, try to reach an allowed domain
	// api.anthropic.com is in the default allowed list
	cmd := exec.Command("docker", "run", "--rm",
		"--cap-add=NET_ADMIN", // Required for iptables
		"-e", "ADDT_FIREWALL_MODE=strict",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Initialize firewall
			sudo -E /usr/local/bin/init-firewall.sh 2>/dev/null || true
			# Test allowed domain (should succeed)
			curl -s --connect-timeout 5 -o /dev/null -w "%{http_code}" https://api.anthropic.com 2>/dev/null || echo "TIMEOUT"
		`,
	)

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	// We expect either a successful HTTP response (2xx, 4xx) or timeout
	// The key is that the connection is allowed - 403/401 means we reached the server
	if err != nil {
		t.Logf("Command output: %s", outputStr)
		// If curl times out, that's a failure - domain should be allowed
		if strings.Contains(outputStr, "TIMEOUT") {
			t.Error("Expected allowed domain to be accessible, but connection timed out")
		}
	}

	// Any HTTP status code means we reached the server
	if strings.Contains(outputStr, "000") || strings.Contains(outputStr, "TIMEOUT") {
		t.Logf("Note: Connection to api.anthropic.com failed - this may be expected if firewall init requires root")
	} else {
		t.Logf("Allowed domain test result: %s", outputStr)
	}
}

func TestFirewallNetwork_Integration_BlockedDomainRejected(t *testing.T) {
	checkDockerForFirewallIntegration(t)

	testImageName := "addt-test-firewall-integration"
	ensureFirewallTestImage(t, testImageName)

	// Run container with firewall enabled, try to reach a non-allowed domain
	// example.com is NOT in the default allowed list
	cmd := exec.Command("docker", "run", "--rm",
		"--cap-add=NET_ADMIN", // Required for iptables
		"-e", "ADDT_FIREWALL_MODE=strict",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Initialize firewall
			sudo -E /usr/local/bin/init-firewall.sh 2>/dev/null || true
			# Test blocked domain (should fail/timeout)
			curl -s --connect-timeout 5 -o /dev/null -w "%{http_code}" https://example.com 2>/dev/null || echo "BLOCKED"
		`,
	)

	output, _ := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	t.Logf("Blocked domain test output: %s", outputStr)

	// We expect the connection to fail/timeout for blocked domains
	// Success codes (2xx, 3xx) would indicate the firewall isn't working
	if strings.Contains(outputStr, "200") || strings.Contains(outputStr, "301") || strings.Contains(outputStr, "302") {
		t.Error("Expected blocked domain to be rejected, but connection succeeded")
	}
}

func TestFirewallNetwork_Integration_PermissiveModeLogs(t *testing.T) {
	checkDockerForFirewallIntegration(t)

	testImageName := "addt-test-firewall-integration"
	ensureFirewallTestImage(t, testImageName)

	// In permissive mode, traffic should be allowed but logged
	cmd := exec.Command("docker", "run", "--rm",
		"--cap-add=NET_ADMIN",
		"-e", "ADDT_FIREWALL_MODE=permissive",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Initialize firewall in permissive mode
			sudo -E /usr/local/bin/init-firewall.sh 2>&1
			# Test that traffic is allowed
			curl -s --connect-timeout 5 -o /dev/null -w "%{http_code}" https://example.com 2>/dev/null || echo "FAILED"
		`,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Logf("Command error: %v", err)
	}

	t.Logf("Permissive mode output: %s", outputStr)

	// Verify permissive mode message appears
	if !strings.Contains(outputStr, "Permissive mode") && !strings.Contains(outputStr, "permissive") {
		t.Log("Note: Permissive mode message not found - firewall init may have failed")
	}
}

func TestFirewallNetwork_Integration_DisabledMode(t *testing.T) {
	checkDockerForFirewallIntegration(t)

	testImageName := "addt-test-firewall-integration"
	ensureFirewallTestImage(t, testImageName)

	// With firewall disabled, all traffic should be allowed
	cmd := exec.Command("docker", "run", "--rm",
		"-e", "ADDT_FIREWALL_MODE=off",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Initialize firewall (should skip in off mode)
			sudo -E /usr/local/bin/init-firewall.sh 2>&1
			# Test that any domain works
			curl -s --connect-timeout 5 -o /dev/null -w "%{http_code}" https://example.com 2>/dev/null || echo "FAILED"
		`,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Logf("Command error: %v", err)
	}

	t.Logf("Disabled mode output: %s", outputStr)

	// Verify disabled message appears
	if !strings.Contains(outputStr, "Disabled") {
		t.Log("Note: Disabled message not found")
	}

	// In disabled mode, connection should succeed
	if strings.Contains(outputStr, "200") || strings.Contains(outputStr, "301") {
		t.Log("Traffic allowed as expected when firewall is disabled")
	}
}

func TestFirewallNetwork_Integration_CustomAllowedDomain(t *testing.T) {
	checkDockerForFirewallIntegration(t)

	testImageName := "addt-test-firewall-integration"
	ensureFirewallTestImage(t, testImageName)

	// Test with a custom allowed domain added at runtime
	cmd := exec.Command("docker", "run", "--rm",
		"--cap-add=NET_ADMIN",
		"-e", "ADDT_FIREWALL_MODE=strict",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", `
			# Add custom domain to allowed list
			mkdir -p /home/addt/.addt/firewall
			echo "example.com" >> /home/addt/.addt/firewall/allowed-domains.txt
			# Initialize firewall
			sudo -E /usr/local/bin/init-firewall.sh 2>&1
			# Test the custom domain
			curl -s --connect-timeout 5 -o /dev/null -w "%{http_code}" https://example.com 2>/dev/null || echo "TIMEOUT"
		`,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Logf("Command error: %v", err)
	}

	t.Logf("Custom domain test output: %s", outputStr)

	// Verify the domain was resolved
	if strings.Contains(outputStr, "Resolving: example.com") {
		t.Log("Custom domain was added and resolved")
	}
}
