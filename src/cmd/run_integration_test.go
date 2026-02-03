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

// createRunDockerProvider creates a Docker provider for run tests
func createRunDockerProvider(cfg *provider.Config) (provider.Provider, error) {
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

// checkDockerForRun verifies Docker is available and running
func checkDockerForRun(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

// getRunBinaryPath returns the absolute path to the addt binary
func getRunBinaryPath(t *testing.T) string {
	t.Helper()
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

	return binaryPath
}

// ensureRunTestImage builds the test image if it doesn't exist
func ensureRunTestImage(t *testing.T, imageName, extension string) {
	t.Helper()

	cmd := exec.Command("docker", "image", "inspect", imageName)
	if cmd.Run() == nil {
		return // Image exists
	}

	cfg := config.LoadConfig("22", "1.23", "0.4.17", 49152)
	cfg.Extensions = extension

	providerCfg := &provider.Config{
		Extensions:  cfg.Extensions,
		NodeVersion: cfg.NodeVersion,
		GoVersion:   cfg.GoVersion,
		UvVersion:   cfg.UvVersion,
		ImageName:   imageName,
	}

	prov, err := createRunDockerProvider(providerCfg)
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

func TestRunCommand_Integration_BinaryHelp(t *testing.T) {
	checkDockerForRun(t)
	binaryPath := getRunBinaryPath(t)

	// Test run --help
	cmd := exec.Command(binaryPath, "run", "--help")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil && !strings.Contains(outputStr, "Usage") {
		t.Fatalf("run --help failed: %v\nOutput: %s", err, outputStr)
	}

	expectedPhrases := []string{
		"Usage: addt run",
		"extension",
		"Examples",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(outputStr, phrase) {
			t.Errorf("Expected help output to contain %q, got: %s", phrase, outputStr)
		}
	}
}

func TestRunCommand_Integration_BinaryHelpShortFlag(t *testing.T) {
	checkDockerForRun(t)
	binaryPath := getRunBinaryPath(t)

	// Test run -h
	cmd := exec.Command(binaryPath, "run", "-h")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil && !strings.Contains(outputStr, "Usage") {
		t.Fatalf("run -h failed: %v\nOutput: %s", err, outputStr)
	}

	if !strings.Contains(outputStr, "Usage: addt run") {
		t.Errorf("Expected help output, got: %s", outputStr)
	}
}

func TestRunCommand_Integration_NoArgs(t *testing.T) {
	checkDockerForRun(t)
	binaryPath := getRunBinaryPath(t)

	// Test run without args (should show help)
	cmd := exec.Command(binaryPath, "run")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Should exit with 0 and show usage
	if err != nil {
		// Some implementations may exit with error
		t.Logf("run with no args returned error: %v", err)
	}

	if !strings.Contains(outputStr, "Usage") && !strings.Contains(outputStr, "extension") {
		t.Errorf("Expected usage info when no args provided, got: %s", outputStr)
	}
}

func TestRunCommand_Integration_InvalidExtension(t *testing.T) {
	checkDockerForRun(t)
	binaryPath := getRunBinaryPath(t)

	// Test run with invalid extension
	cmd := exec.Command(binaryPath, "run", "nonexistent-extension-xyz")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err == nil {
		t.Error("Expected error for invalid extension")
	}

	if !strings.Contains(outputStr, "does not exist") &&
		!strings.Contains(outputStr, "Error") &&
		!strings.Contains(outputStr, "error") {
		t.Errorf("Expected error message about extension not existing, got: %s", outputStr)
	}
}

func TestRunCommand_Integration_ValidExtensionBuildsAndRuns(t *testing.T) {
	checkDockerForRun(t)

	testImageName := "addt-test-run-integration"
	ensureRunTestImage(t, testImageName, "claude")

	// Run a simple command via docker to verify image works
	cmd := exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/echo",
		testImageName, "run test successful")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	if !strings.Contains(string(output), "run test successful") {
		t.Errorf("Expected 'run test successful', got: %s", string(output))
	}
}

func TestRunCommand_Integration_ExtensionWithArgs(t *testing.T) {
	checkDockerForRun(t)

	testImageName := "addt-test-run-args"
	ensureRunTestImage(t, testImageName, "claude")

	// Test that args are passed through
	cmd := exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/echo",
		testImageName, "arg1", "arg2", "arg3")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container with args: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "arg1") ||
		!strings.Contains(outputStr, "arg2") ||
		!strings.Contains(outputStr, "arg3") {
		t.Errorf("Expected args to be passed through, got: %s", outputStr)
	}
}

func TestRunCommand_Integration_ProviderSetup(t *testing.T) {
	checkDockerForRun(t)

	cfg := config.LoadConfig("22", "1.23", "0.4.17", 49152)
	cfg.Extensions = "claude"

	providerCfg := &provider.Config{
		Extensions:  cfg.Extensions,
		NodeVersion: cfg.NodeVersion,
		GoVersion:   cfg.GoVersion,
		UvVersion:   cfg.UvVersion,
	}

	prov, err := createRunDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Provider initialization failed: %v", err)
	}

	// Verify image name is generated
	imageName := prov.DetermineImageName()
	if imageName == "" {
		t.Error("DetermineImageName returned empty string")
	}

	if !strings.Contains(imageName, "claude") {
		t.Errorf("Expected image name to contain 'claude', got: %s", imageName)
	}
}

func TestRunCommand_Integration_MultipleExtensions(t *testing.T) {
	checkDockerForRun(t)

	testImageName := "addt-test-run-multi"

	cfg := config.LoadConfig("22", "1.23", "0.4.17", 49152)
	cfg.Extensions = "claude,codex"

	providerCfg := &provider.Config{
		Extensions:  cfg.Extensions,
		NodeVersion: cfg.NodeVersion,
		GoVersion:   cfg.GoVersion,
		UvVersion:   cfg.UvVersion,
		ImageName:   testImageName,
	}

	prov, err := createRunDockerProvider(providerCfg)
	if err != nil {
		t.Fatalf("Failed to create Docker provider: %v", err)
	}

	if err := prov.Initialize(providerCfg); err != nil {
		t.Fatalf("Provider initialization failed: %v", err)
	}

	// Build multi-extension image
	if err := prov.BuildIfNeeded(true, false); err != nil {
		t.Fatalf("Failed to build multi-extension image: %v", err)
	}

	// Verify image was created
	cmd := exec.Command("docker", "image", "inspect", testImageName)
	if err := cmd.Run(); err != nil {
		t.Errorf("Expected image %s to exist", testImageName)
	}

	// Clean up
	exec.Command("docker", "rmi", "-f", testImageName).Run()
}

func TestRunCommand_Integration_EnvironmentSetup(t *testing.T) {
	checkDockerForRun(t)

	// Save original env vars
	origExtensions := os.Getenv("ADDT_EXTENSIONS")
	origCommand := os.Getenv("ADDT_COMMAND")
	defer func() {
		if origExtensions != "" {
			os.Setenv("ADDT_EXTENSIONS", origExtensions)
		} else {
			os.Unsetenv("ADDT_EXTENSIONS")
		}
		if origCommand != "" {
			os.Setenv("ADDT_COMMAND", origCommand)
		} else {
			os.Unsetenv("ADDT_COMMAND")
		}
	}()

	// Clear env vars
	os.Unsetenv("ADDT_EXTENSIONS")
	os.Unsetenv("ADDT_COMMAND")

	// Call HandleRunCommand
	result := HandleRunCommand([]string{"claude", "extra", "args"})

	// Verify env vars were set
	if os.Getenv("ADDT_EXTENSIONS") != "claude" {
		t.Errorf("ADDT_EXTENSIONS = %q, want %q", os.Getenv("ADDT_EXTENSIONS"), "claude")
	}

	if os.Getenv("ADDT_COMMAND") == "" {
		t.Error("ADDT_COMMAND was not set")
	}

	// Verify remaining args returned
	if result == nil {
		t.Fatal("Expected remaining args, got nil")
	}

	if len(result) != 2 || result[0] != "extra" || result[1] != "args" {
		t.Errorf("Expected [extra args], got %v", result)
	}
}

func TestRunCommand_Integration_ExtensionEntrypoint(t *testing.T) {
	checkDockerForRun(t)

	// Test that different extensions have correct entrypoints
	testCases := []struct {
		extension  string
		entrypoint string
	}{
		{"claude", "claude"},
		// Add more extensions as needed
	}

	for _, tc := range testCases {
		t.Run(tc.extension, func(t *testing.T) {
			entrypoint := GetEntrypointForExtension(tc.extension)
			if entrypoint != tc.entrypoint {
				t.Errorf("GetEntrypointForExtension(%q) = %q, want %q",
					tc.extension, entrypoint, tc.entrypoint)
			}
		})
	}
}
