//go:build integration

package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jedi4ever/addt/assets"
	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/provider/docker"
)

// createShellDockerProvider creates a Docker provider for shell tests
func createShellDockerProvider(cfg *provider.Config) (provider.Provider, error) {
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

// checkDockerForShell verifies Docker is available and running
func checkDockerForShell(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

// containerExists checks if a Docker container exists
func containerExists(containerName string) bool {
	cmd := exec.Command("docker", "container", "inspect", containerName)
	return cmd.Run() == nil
}

// removeContainer removes a Docker container if it exists
func removeContainer(containerName string) {
	exec.Command("docker", "rm", "-f", containerName).Run()
}

// ensureTestImage builds the test image if it doesn't exist
func ensureTestImage(t *testing.T, imageName, extension string) {
	t.Helper()

	// Check if image already exists
	cmd := exec.Command("docker", "image", "inspect", imageName)
	if cmd.Run() == nil {
		return // Image exists
	}

	// Build the image
	cfg := config.LoadConfig("22", "1.23", "0.4.17", 49152)
	cfg.Extensions = extension

	providerCfg := &provider.Config{
		Extensions:  cfg.Extensions,
		NodeVersion: cfg.NodeVersion,
		GoVersion:   cfg.GoVersion,
		UvVersion:   cfg.UvVersion,
		ImageName:   imageName,
	}

	prov, err := createShellDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}

	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("Failed to build test image: %v", err)
	}
}

func TestShellCommand_Integration_RunsContainer(t *testing.T) {
	checkDockerForShell(t)

	testImageName := "addt-test-shell-integration"
	testContainerName := "addt-shell-test-container"

	// Ensure image exists
	ensureTestImage(t, testImageName, "claude")

	// Clean up container before and after
	removeContainer(testContainerName)
	defer removeContainer(testContainerName)

	// Run a simple command in shell mode
	// Use --entrypoint to bypass the claude setup script
	cmd := exec.Command("docker", "run", "--rm", "--name", testContainerName,
		"--entrypoint", "/bin/echo",
		testImageName, "hello from shell")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Shell command failed: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "hello from shell") {
		t.Errorf("Expected output to contain 'hello from shell', got: %s", string(output))
	}
}

func TestShellCommand_Integration_BinaryHelp(t *testing.T) {
	checkDockerForShell(t)

	// Get absolute path to binary
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get working directory: %v", err)
	}

	// Navigate from src/cmd to dist/addt
	binaryPath := wd + "/../../dist/addt"

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try building it
		buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
		buildCmd.Dir = wd + "/.."
		if err := buildCmd.Run(); err != nil {
			t.Skipf("Could not build addt binary: %v, skipping binary test", err)
		}
	}

	// Test shell --help
	cmd := exec.Command(binaryPath, "shell", "--help")
	output, err := cmd.CombinedOutput()

	// --help might exit with code 0 or show help via stdout
	outputStr := string(output)

	if err != nil && !strings.Contains(outputStr, "Usage") {
		t.Fatalf("shell --help failed: %v\nOutput: %s", err, outputStr)
	}

	expectedPhrases := []string{
		"shell",
		"extension",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(outputStr, phrase) {
			t.Errorf("Expected help output to contain %q, got: %s", phrase, outputStr)
		}
	}
}

func TestShellCommand_Integration_NoExtension(t *testing.T) {
	checkDockerForShell(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get working directory: %v", err)
	}
	binaryPath := wd + "/../../dist/addt"

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
		buildCmd.Dir = wd + "/.."
		if err := buildCmd.Run(); err != nil {
			t.Skipf("Could not build addt binary: %v, skipping binary test", err)
		}
	}

	// Test shell without extension (should show help/error)
	cmd := exec.Command(binaryPath, "shell")
	cmd.Env = append(os.Environ(),
		"ADDT_EXTENSIONS=", // Clear any env var
	)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Should exit with error or show usage
	if err == nil {
		// If no error, should still show usage info
		if !strings.Contains(outputStr, "Usage") && !strings.Contains(outputStr, "extension") {
			t.Errorf("Expected usage info when no extension provided, got: %s", outputStr)
		}
	}
}

func TestShellCommand_Integration_InvalidExtension(t *testing.T) {
	checkDockerForShell(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Could not get working directory: %v", err)
	}
	binaryPath := wd + "/../../dist/addt"

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
		buildCmd.Dir = wd + "/.."
		if err := buildCmd.Run(); err != nil {
			t.Skipf("Could not build addt binary: %v, skipping binary test", err)
		}
	}

	// Test shell with invalid extension
	cmd := exec.Command(binaryPath, "shell", "nonexistent-extension-xyz")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err == nil {
		t.Error("Expected error for invalid extension")
	}

	// Check for error message (could be "does not exist", "Error", or exit status)
	if !strings.Contains(outputStr, "does not exist") &&
		!strings.Contains(outputStr, "Error") &&
		!strings.Contains(outputStr, "error") {
		t.Errorf("Expected error message about extension not existing, got: %s", outputStr)
	}
}

func TestShellCommand_Integration_ProviderInitialization(t *testing.T) {
	checkDockerForShell(t)

	testImageName := "addt-test-shell-init"

	cfg := config.LoadConfig("22", "1.23", "0.4.17", 49152)
	cfg.Extensions = "claude"

	providerCfg := &provider.Config{
		Extensions:        cfg.Extensions,
		ExtensionVersions: cfg.ExtensionVersions,
		NodeVersion:       cfg.NodeVersion,
		GoVersion:         cfg.GoVersion,
		UvVersion:         cfg.UvVersion,
		ImageName:         testImageName,
	}

	prov, err := createShellDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	// Test Initialize
	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Provider initialization failed: %v", err)
	}

	// Test DetermineImageName
	imageName := prov.DetermineImageName()
	if imageName == "" {
		t.Error("DetermineImageName returned empty string")
	}
}

func TestShellCommand_Integration_ContainerCleanup(t *testing.T) {
	checkDockerForShell(t)

	testImageName := "addt-test-shell-cleanup"
	testContainerName := "addt-shell-cleanup-test"

	ensureTestImage(t, testImageName, "claude")
	removeContainer(testContainerName)
	defer removeContainer(testContainerName)

	// Start a container that exits quickly
	cmd := exec.Command("docker", "run", "-d", "--name", testContainerName,
		testImageName, "sleep", "2")

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Container should exist
	if !containerExists(testContainerName) {
		t.Error("Container should exist after starting")
	}

	// Wait for container to exit
	time.Sleep(3 * time.Second)

	// Clean up
	removeContainer(testContainerName)

	// Container should be gone
	if containerExists(testContainerName) {
		t.Error("Container should not exist after removal")
	}
}

func TestShellCommand_Integration_EnvironmentVariables(t *testing.T) {
	checkDockerForShell(t)

	testImageName := "addt-test-shell-env"
	ensureTestImage(t, testImageName, "claude")

	// Run container and check environment variables are passed
	// Use --entrypoint to bypass claude setup
	cmd := exec.Command("docker", "run", "--rm",
		"--entrypoint", "/usr/bin/printenv",
		"-e", "TEST_VAR=test_value",
		testImageName, "TEST_VAR")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container with env var: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "test_value") {
		t.Errorf("Expected env var TEST_VAR=test_value, got: %s", string(output))
	}
}

func TestShellCommand_Integration_WorkingDirectory(t *testing.T) {
	checkDockerForShell(t)

	testImageName := "addt-test-shell-workdir"
	ensureTestImage(t, testImageName, "claude")

	// Run container and check working directory
	// Use --entrypoint to bypass claude setup
	cmd := exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/pwd",
		"-w", "/tmp",
		testImageName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container with workdir: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "/tmp") {
		t.Errorf("Expected working directory /tmp, got: %s", string(output))
	}
}
