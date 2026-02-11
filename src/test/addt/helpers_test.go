//go:build addt

package addt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/cmd"
	testutil "github.com/jedi4ever/addt/test/util"
)

// --- Constants ---

const testVersion = testutil.TestVersion

// --- Convenience aliases so test files can use short names ---

var (
	extractMarker              = testutil.ExtractMarker
	availableProviders         = testutil.AvailableProviders
	requireProviders           = testutil.RequireProviders
	requireSSHAgent            = testutil.RequireSSHAgent
	setupAddtDir               = testutil.SetupAddtDir
	setupAddtDirWithExtensions = testutil.SetupAddtDirWithExtensions
	captureOutput              = testutil.CaptureOutput
	runShellCommand            = testutil.RunShellCommand
	runRunSubcommand           = testutil.RunRunSubcommand
	runShellSubcommand         = testutil.RunShellSubcommand
	runContainersSubcommand    = testutil.RunContainersSubcommand
	runAliasCommand            = testutil.RunAliasCommand
	ensureAddtImage            = testutil.EnsureAddtImage
	setDummyAnthropicKey       = testutil.SetDummyAnthropicKey
	procEnvLeakCommand         = testutil.ProcEnvLeakCommand
	runCmd                     = testutil.RunCmd
	saveRestoreEnv             = testutil.SaveRestoreEnv
	requireTmux                = testutil.RequireTmux
	getAddtBinary              = testutil.GetAddtBinary
	requireEnvKey              = testutil.RequireEnvKey
)

// requireUnshareNet skips the test if bwrap --unshare-net doesn't work.
// Some environments (gVisor, restricted containers) can't create new
// network namespaces. Call this inside provider loops for bwrap subtests.
func requireUnshareNet(t *testing.T) {
	t.Helper()
	cmd := exec.Command("bwrap",
		"--unshare-net",
		"--dev", "/dev",
		"--proc", "/proc",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--symlink", "/usr/bin", "/bin",
		"--", "/bin/true")
	if err := cmd.Run(); err != nil {
		t.Skip("bwrap --unshare-net not supported in this environment, skipping")
	}
}

// requireNsenter skips the test if nsenter can't properly join bwrap namespaces.
// Some environments (gVisor) don't fully support namespace operations needed
// for bwrap persistent mode.
func requireNsenter(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("nsenter"); err != nil {
		t.Skip("nsenter not available, skipping")
	}

	testDir := t.TempDir()
	markerPath := filepath.Join(testDir, "nsenter_test")
	os.WriteFile(markerPath, []byte("ok"), 0644)

	bwrapCmd := exec.Command("bwrap",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--symlink", "/usr/bin", "/bin",
		"--bind", testDir, "/run/nstest",
		"--unshare-pid",
		"--", "sleep", "5")
	if err := bwrapCmd.Start(); err != nil {
		t.Skip("can't start bwrap background process, skipping")
	}
	pid := bwrapCmd.Process.Pid
	defer bwrapCmd.Process.Kill()

	nsCmd := exec.Command("nsenter",
		"--target", fmt.Sprintf("%d", pid),
		"--mount",
		"--", "cat", "/run/nstest/nsenter_test")
	out, err := nsCmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Skip("nsenter can't see bwrap bind-mounts in this environment, skipping")
	}
}

// --- Subprocess helpers ---
// cmd.Execute calls os.Exit on errors, so we run it in a subprocess
// (the test binary itself) to avoid killing the test process.

// TestShellHelper is invoked as a subprocess by RunShellCommand.
func TestShellHelper(t *testing.T) {
	ext := os.Getenv("ADDT_TEST_SHELL_EXT")
	if ext == "" {
		t.Skip("not invoked as subprocess")
	}

	command := os.Getenv("ADDT_TEST_SHELL_CMD")
	if command == "" {
		command = "/bin/bash"
	}
	os.Setenv("ADDT_COMMAND", command)
	os.Setenv("ADDT_EXTENSIONS", ext)

	argsStr := os.Getenv("ADDT_TEST_SHELL_ARGS")
	var cliArgs []string
	if argsStr != "" {
		cliArgs = strings.Split(argsStr, "\n")
	}
	os.Args = append([]string{"addt"}, cliArgs...)

	cmd.Execute(testutil.TestVersion, testutil.TestNodeVersion, testutil.TestGoVersion, testutil.TestUvVersion, testutil.TestPortRangeStart)
}

// TestBuildHelper is invoked as a subprocess by EnsureAddtImage.
func TestBuildHelper(t *testing.T) {
	ext := os.Getenv("ADDT_TEST_BUILD_EXT")
	if ext == "" {
		t.Skip("not invoked as subprocess")
	}

	os.Args = []string{"addt", "build", ext}
	cmd.Execute(testutil.TestVersion, testutil.TestNodeVersion, testutil.TestGoVersion, testutil.TestUvVersion, testutil.TestPortRangeStart)
}

// TestRunSubcommandHelper is invoked as a subprocess by RunRunSubcommand.
func TestRunSubcommandHelper(t *testing.T) {
	ext := os.Getenv("ADDT_TEST_RUNSUB_EXT")
	if ext == "" {
		t.Skip("not invoked as subprocess")
	}

	osArgs := []string{"addt", "run", ext}
	argsStr := os.Getenv("ADDT_TEST_RUNSUB_ARGS")
	if argsStr != "" {
		osArgs = append(osArgs, strings.Split(argsStr, "\n")...)
	}
	os.Args = osArgs

	cmd.Execute(testutil.TestVersion, testutil.TestNodeVersion, testutil.TestGoVersion, testutil.TestUvVersion, testutil.TestPortRangeStart)
}

// TestShellSubcommandHelper is invoked as a subprocess by RunShellSubcommand.
func TestShellSubcommandHelper(t *testing.T) {
	ext := os.Getenv("ADDT_TEST_SHELLSUB_EXT")
	if ext == "" {
		t.Skip("not invoked as subprocess")
	}

	osArgs := []string{"addt", "shell", ext}
	argsStr := os.Getenv("ADDT_TEST_SHELLSUB_ARGS")
	if argsStr != "" {
		osArgs = append(osArgs, strings.Split(argsStr, "\n")...)
	}
	os.Args = osArgs

	cmd.Execute(testutil.TestVersion, testutil.TestNodeVersion, testutil.TestGoVersion, testutil.TestUvVersion, testutil.TestPortRangeStart)
}

// TestContainersSubcommandHelper is invoked as a subprocess by RunContainersSubcommand.
func TestContainersSubcommandHelper(t *testing.T) {
	argsStr := os.Getenv("ADDT_TEST_CONTAINERS_ARGS")
	if argsStr == "" {
		t.Skip("not invoked as subprocess")
	}

	osArgs := []string{"addt", "containers"}
	osArgs = append(osArgs, strings.Split(argsStr, "\n")...)
	os.Args = osArgs

	cmd.Execute(testutil.TestVersion, testutil.TestNodeVersion, testutil.TestGoVersion, testutil.TestUvVersion, testutil.TestPortRangeStart)
}

// TestAliasHelper is invoked as a subprocess by RunAliasCommand.
func TestAliasHelper(t *testing.T) {
	alias := os.Getenv("ADDT_TEST_ALIAS_NAME")
	if alias == "" {
		t.Skip("not invoked as subprocess")
	}

	osArgs := []string{"addt-" + alias}
	argsStr := os.Getenv("ADDT_TEST_ALIAS_ARGS")
	if argsStr != "" {
		osArgs = append(osArgs, strings.Split(argsStr, "\n")...)
	}
	os.Args = osArgs

	cmd.Execute(testutil.TestVersion, testutil.TestNodeVersion, testutil.TestGoVersion, testutil.TestUvVersion, testutil.TestPortRangeStart)
}
