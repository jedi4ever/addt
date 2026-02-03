//go:build integration

package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/provider/docker"
)

// createDockerProvider creates a Docker provider with embedded assets
func createDockerProvider(cfg *provider.Config) (provider.Provider, error) {
	return docker.NewDockerProvider(
		cfg,
		assets.DockerDockerfile,
		assets.DockerDockerfileBase,
		assets.DockerEntrypoint,
		assets.DockerInitFirewall,
		assets.DockerInstallSh,
		extensions.FS,
	)
}

// checkDocker verifies Docker is available and running
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

// imageExists checks if a Docker image exists
func imageExists(imageName string) bool {
	cmd := exec.Command("docker", "image", "inspect", imageName)
	return cmd.Run() == nil
}

// removeImage removes a Docker image if it exists
func removeImage(imageName string) {
	exec.Command("docker", "rmi", "-f", imageName).Run()
}

func TestBuildCommand_Integration_Claude(t *testing.T) {
	checkDocker(t)

	// Use a test-specific image name to avoid conflicts
	testImageName := "addt-test-claude-integration"

	// Clean up before and after test
	removeImage(testImageName)
	defer removeImage(testImageName)

	// Load config with defaults
	cfg := config.LoadConfig("22", "1.23.5", "0.4.17", 49152)
	cfg.Extensions = "claude"

	// Create provider config
	providerCfg := &provider.Config{
		Extensions:        cfg.Extensions,
		ExtensionVersions: cfg.ExtensionVersions,
		NodeVersion:       cfg.NodeVersion,
		GoVersion:         cfg.GoVersion,
		UvVersion:         cfg.UvVersion,
		ImageName:         testImageName,
	}

	// Create Docker provider
	prov, err := createDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	// Initialize provider
	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	// Build the image
	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("BuildIfNeeded failed: %v", err)
	}

	// Verify image was created
	if !imageExists(testImageName) {
		t.Error("Expected image to exist after build")
	}
}

func TestBuildCommand_Integration_WithNoCache(t *testing.T) {
	checkDocker(t)

	testImageName := "addt-test-nocache-integration"

	removeImage(testImageName)
	defer removeImage(testImageName)

	cfg := config.LoadConfig("22", "1.23.5", "0.4.17", 49152)
	cfg.Extensions = "claude"

	providerCfg := &provider.Config{
		Extensions:        cfg.Extensions,
		ExtensionVersions: cfg.ExtensionVersions,
		NodeVersion:       cfg.NodeVersion,
		GoVersion:         cfg.GoVersion,
		UvVersion:         cfg.UvVersion,
		ImageName:         testImageName,
		NoCache:           true, // Test no-cache flag
	}

	prov, err := createDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("BuildIfNeeded with NoCache failed: %v", err)
	}

	if !imageExists(testImageName) {
		t.Error("Expected image to exist after no-cache build")
	}
}

func TestBuildCommand_Integration_Binary(t *testing.T) {
	checkDocker(t)

	// Find the built binary
	binaryPath := "../dist/addt"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try building it
		buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
		buildCmd.Dir = ".."
		if err := buildCmd.Run(); err != nil {
			t.Skip("Could not build addt binary, skipping binary test")
		}
	}

	testImageName := "addt-test-binary-integration"
	removeImage(testImageName)
	defer removeImage(testImageName)

	// Run the actual binary
	cmd := exec.Command(binaryPath, "build", "claude")
	cmd.Env = append(os.Environ(),
		"ADDT_IMAGE_NAME="+testImageName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("addt build command failed: %v\nOutput: %s", err, string(output))
	}

	if !imageExists(testImageName) {
		t.Errorf("Expected image %s to exist after binary build\nOutput: %s", testImageName, string(output))
	}
}

func TestBuildCommand_Integration_ExtensionVersion(t *testing.T) {
	checkDocker(t)

	testImageName := "addt-test-version-integration"

	removeImage(testImageName)
	defer removeImage(testImageName)

	cfg := config.LoadConfig("22", "1.23.5", "0.4.17", 49152)
	cfg.Extensions = "claude"

	// Set a specific version
	providerCfg := &provider.Config{
		Extensions: cfg.Extensions,
		ExtensionVersions: map[string]string{
			"claude": "1.0.21", // Use a specific known version
		},
		NodeVersion: cfg.NodeVersion,
		GoVersion:   cfg.GoVersion,
		UvVersion:   cfg.UvVersion,
		ImageName:   testImageName,
	}

	prov, err := createDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("BuildIfNeeded with specific version failed: %v", err)
	}

	if !imageExists(testImageName) {
		t.Error("Expected image to exist after versioned build")
	}

	// Verify the version is in the image labels or env
	cmd := exec.Command("docker", "inspect", "--format", "{{.Config.Labels}}", testImageName)
	output, err := cmd.Output()
	if err == nil {
		// Check if version is mentioned (implementation-dependent)
		t.Logf("Image labels: %s", string(output))
	}
}

func TestBuildCommand_Integration_MultipleExtensions(t *testing.T) {
	checkDocker(t)

	testImageName := "addt-test-multi-integration"

	removeImage(testImageName)
	defer removeImage(testImageName)

	cfg := config.LoadConfig("22", "1.23.5", "0.4.17", 49152)
	cfg.Extensions = "claude,codex"

	providerCfg := &provider.Config{
		Extensions:        cfg.Extensions,
		ExtensionVersions: cfg.ExtensionVersions,
		NodeVersion:       cfg.NodeVersion,
		GoVersion:         cfg.GoVersion,
		UvVersion:         cfg.UvVersion,
		ImageName:         testImageName,
	}

	prov, err := createDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("BuildIfNeeded with multiple extensions failed: %v", err)
	}

	if !imageExists(testImageName) {
		t.Error("Expected image to exist after multi-extension build")
	}
}

func TestBuildCommand_Integration_InvalidExtension(t *testing.T) {
	checkDocker(t)

	cfg := config.LoadConfig("22", "1.23.5", "0.4.17", 49152)
	cfg.Extensions = "nonexistent-extension-xyz"

	providerCfg := &provider.Config{
		Extensions:  cfg.Extensions,
		NodeVersion: cfg.NodeVersion,
		GoVersion:   cfg.GoVersion,
		UvVersion:   cfg.UvVersion,
		ImageName:   "addt-test-invalid",
	}

	prov, err := createDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	// Build should fail for invalid extension
	err = prov.BuildIfNeeded(true, false)
	if err == nil {
		t.Error("Expected build to fail for invalid extension")
		removeImage("addt-test-invalid")
	}
}

func TestBuildCommand_Integration_ImageNameFormat(t *testing.T) {
	checkDocker(t)

	cfg := config.LoadConfig("22", "1.23.5", "0.4.17", 49152)
	cfg.Extensions = "claude"

	providerCfg := &provider.Config{
		Extensions:  cfg.Extensions,
		NodeVersion: cfg.NodeVersion,
		GoVersion:   cfg.GoVersion,
		UvVersion:   cfg.UvVersion,
	}

	prov, err := createDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	// Test DetermineImageName returns expected format
	imageName := prov.DetermineImageName()

	if imageName == "" {
		t.Error("DetermineImageName returned empty string")
	}

	// Image name should contain the extension name
	if !strings.Contains(imageName, "claude") {
		t.Errorf("Expected image name to contain 'claude', got: %s", imageName)
	}

	t.Logf("Generated image name: %s", imageName)
}
