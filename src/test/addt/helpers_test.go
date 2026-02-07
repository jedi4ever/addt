//go:build addt

package addt

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/cmd"
)

const (
	testVersion        = "0.0.0-test"
	testNodeVersion    = "22"
	testGoVersion      = "latest"
	testUvVersion      = "latest"
	testPortRangeStart = 30000
)

// --- Subprocess helpers ---
// cmd.Execute calls os.Exit on errors, so we run it in a subprocess
// (the test binary itself) to avoid killing the test process.

// TestShellHelper is invoked as a subprocess by runShellCommand.
// It sets ADDT_COMMAND (default: /bin/bash) and ADDT_EXTENSIONS, then
// calls Execute() which goes through the full entrypoint initialization
// (SSH proxy, secrets, firewall, etc.) before running the command.
func TestShellHelper(t *testing.T) {
	ext := os.Getenv("ADDT_TEST_SHELL_EXT")
	if ext == "" {
		t.Skip("not invoked as subprocess")
	}

	// Set ADDT_COMMAND so the entrypoint runs bash (not the extension)
	command := os.Getenv("ADDT_TEST_SHELL_CMD")
	if command == "" {
		command = "/bin/bash"
	}
	os.Setenv("ADDT_COMMAND", command)
	os.Setenv("ADDT_EXTENSIONS", ext)

	// Build os.Args to simulate CLI invocation via the run path
	argsStr := os.Getenv("ADDT_TEST_SHELL_ARGS")
	var cliArgs []string
	if argsStr != "" {
		cliArgs = strings.Split(argsStr, "\n")
	}
	os.Args = append([]string{"addt"}, cliArgs...)

	cmd.Execute(testVersion, testNodeVersion, testGoVersion, testUvVersion, testPortRangeStart)
}

// TestBuildHelper is invoked as a subprocess by ensureAddtImage.
// It is skipped unless the ADDT_TEST_BUILD_EXT env var is set.
func TestBuildHelper(t *testing.T) {
	ext := os.Getenv("ADDT_TEST_BUILD_EXT")
	if ext == "" {
		t.Skip("not invoked as subprocess")
	}

	os.Args = []string{"addt", "build", ext}
	cmd.Execute(testVersion, testNodeVersion, testGoVersion, testUvVersion, testPortRangeStart)
}

// --- Provider detection ---

// availableProviders returns container providers available on this machine.
func availableProviders(t *testing.T) []string {
	t.Helper()
	var providers []string

	if path, err := exec.LookPath("docker"); err == nil {
		c := exec.Command(path, "info")
		if c.Run() == nil {
			providers = append(providers, "docker")
		}
	}

	if path, err := exec.LookPath("podman"); err == nil {
		c := exec.Command(path, "version")
		if c.Run() == nil {
			if runtime.GOOS == "darwin" {
				mc := exec.Command(path, "machine", "list", "--format", "{{.Running}}")
				out, err := mc.Output()
				if err == nil && strings.Contains(string(out), "true") {
					providers = append(providers, "podman")
				}
			} else {
				providers = append(providers, "podman")
			}
		}
	}

	return providers
}

func requireProviders(t *testing.T) []string {
	t.Helper()
	provs := availableProviders(t)
	if len(provs) == 0 {
		t.Skip("No container provider (docker/podman) available, skipping")
	}
	return provs
}

func requireSSHAgent(t *testing.T) {
	t.Helper()
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Skip("SSH_AUTH_SOCK not set, skipping")
	}
}

// --- Setup and execution helpers ---

// setupAddtDir creates a temp directory with .addt.yaml and isolated
// ADDT_CONFIG_DIR. Sets ADDT_PROVIDER and changes cwd (for in-process calls).
// Returns projectDir and cleanup function.
func setupAddtDir(t *testing.T, provider, yamlContent string) (string, func()) {
	t.Helper()

	projectDir := t.TempDir()
	globalDir := t.TempDir()

	configPath := filepath.Join(projectDir, ".addt.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("Failed to write .addt.yaml: %v", err)
	}

	origConfigDir := os.Getenv("ADDT_CONFIG_DIR")
	origProvider := os.Getenv("ADDT_PROVIDER")
	origCwd, _ := os.Getwd()

	os.Setenv("ADDT_CONFIG_DIR", globalDir)
	os.Setenv("ADDT_PROVIDER", provider)
	os.Chdir(projectDir)

	cleanup := func() {
		if origConfigDir != "" {
			os.Setenv("ADDT_CONFIG_DIR", origConfigDir)
		} else {
			os.Unsetenv("ADDT_CONFIG_DIR")
		}
		if origProvider != "" {
			os.Setenv("ADDT_PROVIDER", origProvider)
		} else {
			os.Unsetenv("ADDT_PROVIDER")
		}
		os.Chdir(origCwd)
	}

	return projectDir, cleanup
}

// captureOutput captures combined stdout+stderr while running fn in-process.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stdout = w
	os.Stderr = w

	outCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		buf.ReadFrom(r)
		outCh <- buf.String()
	}()

	fn()

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return <-outCh
}

// runShellCommand runs a command inside the container via the Execute() run path.
// It sets ADDT_COMMAND=/bin/bash and goes through the entrypoint, so SSH proxy,
// secrets, etc. are properly initialized.
// The first arg is the extension name; the rest are passed as CLI args
// (typically: "-c", "command string").
func runShellCommand(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	ext := args[0]
	cliArgs := args[1:]

	c := exec.Command(os.Args[0], "-test.run=^TestShellHelper$", "-test.v")
	c.Dir = dir
	c.Env = append(os.Environ(),
		"ADDT_TEST_SHELL_EXT="+ext,
		"ADDT_TEST_SHELL_ARGS="+strings.Join(cliArgs, "\n"),
	)
	output, err := c.CombinedOutput()
	return string(output), err
}

// ensureAddtImage builds the extension image via TestBuildHelper subprocess.
func ensureAddtImage(t *testing.T, dir, extension string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping in short mode (image build required)")
	}
	c := exec.Command(os.Args[0], "-test.run=^TestBuildHelper$", "-test.v")
	c.Dir = dir
	c.Env = append(os.Environ(), "ADDT_TEST_BUILD_EXT="+extension)
	output, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("addt build %s failed: %v\nOutput: %s", extension, err, string(output))
	}
	t.Logf("Build output: %s", string(output))
}
