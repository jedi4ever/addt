package testutil

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/provider"
)

const (
	TestVersion        = "0.0.0-test"
	TestNodeVersion    = "22"
	TestGoVersion      = "latest"
	TestUvVersion      = "latest"
	TestPortRangeStart = 30000
)

// --- Marker extraction ---

// ExtractMarker finds a line starting with the given marker prefix and returns the suffix.
// Used across all test files to extract structured results from subprocess output.
// Example: ExtractMarker(output, "SHELL_TEST:") returns "hello" from a line "SHELL_TEST:hello".
func ExtractMarker(output, marker string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, marker) {
			return strings.TrimPrefix(line, marker)
		}
	}
	return ""
}

// ProcEnvLeakCommand returns a shell command that checks whether a given
// env var name appears in /proc/1/environ. Outputs "PROC_RESULT:LEAKED"
// or "PROC_RESULT:ISOLATED".
func ProcEnvLeakCommand(envVar string) string {
	return "if grep -q " + envVar + " /proc/1/environ 2>/dev/null; then echo PROC_RESULT:LEAKED; else echo PROC_RESULT:ISOLATED; fi"
}

// --- Provider detection ---

// AvailableProviders returns container providers available on this machine.
func AvailableProviders(t *testing.T) []string {
	t.Helper()
	var providers []string
	contexts := provider.DockerContextNames()

	// Check Docker Desktop (desktop-linux context)
	if contains(contexts, "desktop-linux") {
		providers = append(providers, "docker")
	}
	// Check Rancher Desktop (rancher-desktop context)
	if contains(contexts, "rancher-desktop") {
		providers = append(providers, "rancher")
	}
	// Check OrbStack (orbstack context, macOS only)
	if runtime.GOOS == "darwin" && contains(contexts, "orbstack") {
		providers = append(providers, "orbstack")
	}
	// Podman (own binary)
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

	// Bwrap (Linux only, no container runtime needed)
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("bwrap"); err == nil {
			providers = append(providers, "bwrap")
		}
	}

	return providers
}

// RequireBwrap skips the test if bubblewrap is not available (Linux + bwrap binary).
func RequireBwrap(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("bwrap requires Linux, skipping")
	}
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not installed, skipping")
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// RequireProviders skips the test if no container providers are available.
func RequireProviders(t *testing.T) []string {
	t.Helper()
	provs := AvailableProviders(t)
	if len(provs) == 0 {
		t.Skip("No container provider (docker/podman/orbstack) available, skipping")
	}
	return provs
}

// RequireSSHAgent skips the test if SSH_AUTH_SOCK is not set.
func RequireSSHAgent(t *testing.T) {
	t.Helper()
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Skip("SSH_AUTH_SOCK not set, skipping")
	}
}

// --- Setup and execution helpers ---

// SetupAddtDir creates a temp directory with .addt.yaml and isolated
// ADDT_CONFIG_DIR. Sets ADDT_PROVIDER and changes cwd (for in-process calls).
// Returns projectDir and cleanup function.
func SetupAddtDir(t *testing.T, provider, yamlContent string) (string, func()) {
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

// SetupAddtDirWithExtensions is like SetupAddtDir but also sets ADDT_EXTENSIONS_DIR
// to point at the testdata/extensions directory containing the debug extension.
func SetupAddtDirWithExtensions(t *testing.T, provider, yamlContent string) (string, func()) {
	t.Helper()

	projectDir, baseCleanup := SetupAddtDir(t, provider, yamlContent)

	// Resolve testdata extensions dir relative to this file:
	// this file is at test/util/helpers.go → go up to test/ → join testdata/extensions
	_, thisFile, _, _ := runtime.Caller(0)
	testdataExtsDir := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "testdata", "extensions")

	origExtsDir := os.Getenv("ADDT_EXTENSIONS_DIR")
	os.Setenv("ADDT_EXTENSIONS_DIR", testdataExtsDir)

	cleanup := func() {
		if origExtsDir != "" {
			os.Setenv("ADDT_EXTENSIONS_DIR", origExtsDir)
		} else {
			os.Unsetenv("ADDT_EXTENSIONS_DIR")
		}
		baseCleanup()
	}

	return projectDir, cleanup
}

// CaptureOutput captures combined stdout+stderr while running fn in-process.
func CaptureOutput(t *testing.T, fn func()) string {
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

// RunShellCommand runs a command inside the container via the Execute() run path.
// It sets ADDT_COMMAND=/bin/bash and goes through the entrypoint, so SSH proxy,
// secrets, etc. are properly initialized.
// The first arg is the extension name; the rest are passed as CLI args
// (typically: "-c", "command string").
func RunShellCommand(t *testing.T, dir string, args ...string) (string, error) {
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

// RunRunSubcommand runs a command via the "addt run" subcommand path.
// This goes through HandleRunCommand → runner.Run → provider.Run,
// which sets ADDT_COMMAND to the extension's entrypoint.
// The first arg is the extension name; the rest are passed as run args.
func RunRunSubcommand(t *testing.T, dir string, ext string, args ...string) (string, error) {
	t.Helper()

	c := exec.Command(os.Args[0], "-test.run=^TestRunSubcommandHelper$", "-test.v")
	c.Dir = dir
	c.Env = append(os.Environ(),
		"ADDT_TEST_RUNSUB_EXT="+ext,
		"ADDT_TEST_RUNSUB_ARGS="+strings.Join(args, "\n"),
	)
	output, err := c.CombinedOutput()
	return string(output), err
}

// RunShellSubcommand runs a command via the "addt shell" subcommand path.
// Unlike RunShellCommand (which uses the run path with ADDT_COMMAND=/bin/bash),
// this goes through HandleShellCommand → runner.Shell → provider.Shell.
// The first arg is the extension name; the rest are passed as shell args.
func RunShellSubcommand(t *testing.T, dir string, ext string, args ...string) (string, error) {
	t.Helper()

	c := exec.Command(os.Args[0], "-test.run=^TestShellSubcommandHelper$", "-test.v")
	c.Dir = dir
	c.Env = append(os.Environ(),
		"ADDT_TEST_SHELLSUB_EXT="+ext,
		"ADDT_TEST_SHELLSUB_ARGS="+strings.Join(args, "\n"),
	)
	output, err := c.CombinedOutput()
	return string(output), err
}

// RunContainersSubcommand runs "addt containers <args...>" via subprocess.
func RunContainersSubcommand(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	c := exec.Command(os.Args[0], "-test.run=^TestContainersSubcommandHelper$", "-test.v")
	c.Dir = dir
	c.Env = append(os.Environ(),
		"ADDT_TEST_CONTAINERS_ARGS="+strings.Join(args, "\n"),
	)
	output, err := c.CombinedOutput()
	return string(output), err
}

// RunAliasCommand runs addt as if invoked via a symlink alias (e.g., "addt-codex").
// The aliasName is the extension name (e.g., "codex"); args are the CLI arguments
// that follow the binary name.
func RunAliasCommand(t *testing.T, dir string, aliasName string, args ...string) (string, error) {
	t.Helper()

	c := exec.Command(os.Args[0], "-test.run=^TestAliasHelper$", "-test.v")
	c.Dir = dir
	c.Env = append(os.Environ(),
		"ADDT_TEST_ALIAS_NAME="+aliasName,
		"ADDT_TEST_ALIAS_ARGS="+strings.Join(args, "\n"),
	)
	output, err := c.CombinedOutput()
	return string(output), err
}

// EnsureAddtImage builds the extension image via TestBuildHelper subprocess.
func EnsureAddtImage(t *testing.T, dir, extension string) {
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

// SetDummyAnthropicKey sets a dummy ANTHROPIC_API_KEY so that the claude
// extension's credentials.sh exits early without hitting the macOS keychain.
// Returns a cleanup function that restores the original value.
func SetDummyAnthropicKey(t *testing.T) func() {
	t.Helper()
	orig := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-dummy-key-for-addt-tests")
	return func() {
		if orig != "" {
			os.Setenv("ANTHROPIC_API_KEY", orig)
		} else {
			os.Unsetenv("ANTHROPIC_API_KEY")
		}
	}
}

// SaveRestoreEnv saves the current value of an environment variable, sets a new
// value, and returns a cleanup function that restores the original. Use this for
// env vars that must be visible to subprocesses (where t.Setenv doesn't work).
func SaveRestoreEnv(t *testing.T, key, newValue string) func() {
	t.Helper()
	orig := os.Getenv(key)
	os.Setenv(key, newValue)
	return func() {
		if orig != "" {
			os.Setenv(key, orig)
		} else {
			os.Unsetenv(key)
		}
	}
}

// RequireTmux skips the test if tmux is not installed. Returns the tmux binary path.
func RequireTmux(t *testing.T) string {
	t.Helper()
	tmuxBin, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux not installed, skipping")
	}
	return tmuxBin
}

// GetAddtBinary returns the path to dist/addt, always rebuilding from source
// to ensure tests run against the current code.
func GetAddtBinary(t *testing.T) string {
	t.Helper()

	// Resolve repo root: this file is at src/test/util/helpers.go
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	binaryPath := filepath.Join(repoRoot, "dist", "addt")

	srcDir := filepath.Join(repoRoot, "src")
	os.MkdirAll(filepath.Dir(binaryPath), 0755)
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = srcDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build addt binary: %v\n%s", err, string(output))
	}

	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("Failed to resolve absolute path for binary: %v", err)
	}
	return absPath
}

// RequireEnvKey checks os.Getenv(key); if empty, tries to load .env.addt
// from the current directory, then from the repo root.
// Skips the test if still empty. Returns the key value.
func RequireEnvKey(t *testing.T, key string) string {
	t.Helper()

	if v := os.Getenv(key); v != "" {
		return v
	}

	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")

	// Try loading .env.addt from current directory, then repo root
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, ".env.addt"),
		filepath.Join(repoRoot, ".env.addt"),
	}

	for _, f := range candidates {
		if _, err := os.Stat(f); err == nil {
			_ = config.LoadEnvFile(f)
		}
		if v := os.Getenv(key); v != "" {
			return v
		}
	}

	t.Skipf("%s not set (checked env and %v), skipping", key, candidates)
	return ""
}

// RunCmd executes a host command and returns trimmed stdout, or empty on error.
func RunCmd(t *testing.T, name string, args ...string) string {
	t.Helper()
	c := exec.Command(name, args...)
	out, err := c.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
