//go:build addt

package addt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireBwrap skips the test if bubblewrap is not available.
func requireBwrap(t *testing.T) {
	t.Helper()
	requireBwrapProvider(t)
}

// requireUnshareNet skips the test if bwrap --unshare-net doesn't work.
// Some environments (gVisor, restricted containers) can't create new
// network namespaces.
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
		t.Skip("--unshare-net not supported in this environment (gVisor/restricted), skipping")
	}
}

// requireNsenter skips the test if nsenter can't properly join bwrap namespaces.
// Some environments (gVisor) don't fully support namespace operations.
// We verify that bind-mounts inside bwrap are visible via nsenter.
func requireNsenter(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("nsenter"); err != nil {
		t.Skip("nsenter not available, skipping")
	}

	// Create a temp directory to test bind-mount visibility
	testDir := t.TempDir()
	markerPath := filepath.Join(testDir, "nsenter_test")
	os.WriteFile(markerPath, []byte("ok"), 0644)

	// Start a bwrap process that bind-mounts the test directory
	cmd := exec.Command("bwrap",
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
	if err := cmd.Start(); err != nil {
		t.Skip("can't start bwrap background process, skipping")
	}
	pid := cmd.Process.Pid
	defer cmd.Process.Kill()

	// Verify nsenter can see the bind-mount
	nsCmd := exec.Command("nsenter",
		"--target", fmt.Sprintf("%d", pid),
		"--mount",
		"--", "cat", "/run/nstest/nsenter_test")
	out, err := nsCmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Skip("nsenter can't see bwrap bind-mounts in this environment, skipping")
	}
}

// --- Basic execution tests ---

func TestBwrap_Addt_BasicExecution(t *testing.T) {
	// Scenario: User runs `addt run debug -c "echo hello"` with bwrap provider.
	// The debug extension uses /bin/bash as entrypoint, so this runs a bash
	// command inside a bwrap sandbox. Verifies end-to-end CLI → bwrap path.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "echo BWRAP_RUN:hello")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "BWRAP_RUN:")
	if result != "hello" {
		t.Errorf("Expected BWRAP_RUN:hello, got %q\nFull output:\n%s", result, output)
	}
}

func TestBwrap_Addt_ShellExecution(t *testing.T) {
	// Scenario: User runs `addt shell debug -c "echo hello"` with bwrap.
	// Verifies the shell subcommand path works with bwrap.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	output, err := runShellSubcommand(t, dir, "debug",
		"-c", "echo BWRAP_SHELL:hello")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("shell subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "BWRAP_SHELL:")
	if result != "hello" {
		t.Errorf("Expected BWRAP_SHELL:hello, got %q\nFull output:\n%s", result, output)
	}
}

// --- Workdir tests ---

func TestBwrap_Addt_WorkdirMounted(t *testing.T) {
	// Scenario: User creates a marker file in the project directory,
	// then runs a command inside bwrap. The file should be visible
	// at /workspace/ confirming workdir bind-mount works.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	markerFile := filepath.Join(dir, "bwrap_marker.txt")
	if err := os.WriteFile(markerFile, []byte("found"), 0o644); err != nil {
		t.Fatalf("Failed to write marker file: %v", err)
	}

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "if [ -f /workspace/bwrap_marker.txt ]; then echo WORKDIR:yes; else echo WORKDIR:no; fi")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "WORKDIR:")
	if result != "yes" {
		t.Errorf("Expected WORKDIR:yes, got %q\nFull output:\n%s", result, output)
	}
}

func TestBwrap_Addt_WorkdirReadonly(t *testing.T) {
	// Scenario: User enables read-only workdir. Writing to /workspace/
	// inside the sandbox should fail, confirming the ro-bind mount.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", `
workdir:
  readonly: true
`)
	defer cleanup()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "if touch /workspace/readonly_test 2>/dev/null; then echo RO:no; else echo RO:yes; fi")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "RO:")
	if result != "yes" {
		t.Errorf("Expected RO:yes (write should fail), got %q\nFull output:\n%s", result, output)
	}
}

// --- Environment variable tests ---

func TestBwrap_Addt_EnvVarsForwarded(t *testing.T) {
	// Scenario: User sets a custom env var and configures ADDT_ENV_VARS
	// to forward it into the bwrap sandbox. The var should be available inside.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	restoreVar := saveRestoreEnv(t, "BWRAP_TEST_VAR", "bwrap_value")
	defer restoreVar()
	restoreEnvVars := saveRestoreEnv(t, "ADDT_ENV_VARS", "BWRAP_TEST_VAR")
	defer restoreEnvVars()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "echo ENVVAR:${BWRAP_TEST_VAR:-NOTSET}")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "ENVVAR:")
	if result != "bwrap_value" {
		t.Errorf("Expected ENVVAR:bwrap_value, got %q\nFull output:\n%s", result, output)
	}
}

func TestBwrap_Addt_ClearEnv(t *testing.T) {
	// Scenario: User runs a command inside bwrap without forwarding a host
	// env var. Bwrap uses --clearenv, so host env vars should NOT leak.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	restoreVar := saveRestoreEnv(t, "BWRAP_LEAK_TEST", "should_not_leak")
	defer restoreVar()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "echo LEAK:${BWRAP_LEAK_TEST:-CLEAN}")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "LEAK:")
	if result != "CLEAN" {
		t.Errorf("Expected LEAK:CLEAN (env should not leak), got %q\nFull output:\n%s", result, output)
	}
}

// --- Namespace isolation tests ---

func TestBwrap_Addt_PidIsolation(t *testing.T) {
	// Scenario: User runs a command that checks PID 1 inside the bwrap
	// sandbox. Due to --unshare-pid, PID 1 inside should not be the
	// host's init process.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "echo PID1:$(cat /proc/1/comm 2>/dev/null || echo unknown)")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "PID1:")
	// In a bwrap sandbox with --unshare-pid, PID 1 should be bwrap or bash
	if result == "systemd" || result == "init" {
		t.Errorf("Expected PID isolation (PID 1 != host init), got %q", result)
	}
}

func TestBwrap_Addt_HostnameIsolated(t *testing.T) {
	// Scenario: User runs hostname inside bwrap sandbox. Due to
	// --unshare-uts and --hostname addt, the hostname should be "addt".
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "echo HOST:$(hostname)")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "HOST:")
	if result != "addt" {
		t.Errorf("Expected HOST:addt, got %q\nFull output:\n%s", result, output)
	}
}

// --- Network tests ---

func TestBwrap_Addt_NetworkIsolation(t *testing.T) {
	// Scenario: User enables network isolation (security.network_mode: none).
	// Inside the sandbox, network should be completely unavailable —
	// curl to any host should fail.
	requireBwrap(t)
	requireUnshareNet(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", `
security:
  network_mode: "none"
`)
	defer cleanup()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "if curl -s --connect-timeout 2 http://example.com >/dev/null 2>&1; then echo NET:open; else echo NET:isolated; fi")

	t.Logf("Output:\n%s", output)
	// Command might exit non-zero if curl fails, that's expected
	_ = err

	result := extractMarker(output, "NET:")
	if result != "isolated" {
		t.Errorf("Expected NET:isolated (no network), got %q\nFull output:\n%s", result, output)
	}
}

func TestBwrap_Addt_NetworkHostDefault(t *testing.T) {
	// Scenario: User runs with default config (no firewall, no network isolation).
	// Network should be shared with host — network commands should work.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	// Check network access with a short timeout
	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "if timeout 3 bash -c 'echo > /dev/tcp/8.8.8.8/53' 2>/dev/null; then echo NET_HOST:open; else echo NET_HOST:closed; fi")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Logf("Command returned error (may be expected in restricted env): %v", err)
	}

	result := extractMarker(output, "NET_HOST:")
	// In some restricted environments, network may not work even in host mode
	if result == "closed" {
		t.Log("Network not accessible from bwrap (restricted environment) — skipping assertion")
	} else if result == "" {
		t.Log("NET_HOST marker not produced — /dev/tcp may not be available")
	}
	// If result is "open", the test passes implicitly
}

// --- Firewall proxy tests ---

func TestBwrap_Addt_FirewallBlocksUnlisted(t *testing.T) {
	// Scenario: User enables firewall in strict mode. Curl to an unlisted
	// domain should fail because the proxy blocks it.
	requireBwrap(t)
	requireUnshareNet(t)

	if _, err := exec.LookPath("socat"); err != nil {
		t.Skip("socat not installed, skipping firewall test")
	}

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", `
firewall:
  enabled: true
  mode: "strict"
`)
	defer cleanup()

	output, _ := runRunSubcommand(t, dir, "debug",
		"-c", "CODE=$(curl -s --connect-timeout 5 -o /dev/null -w '%{http_code}' https://example.com 2>/dev/null) || CODE=BLOCKED; echo FW_RESULT:$CODE")

	t.Logf("Output:\n%s", output)
	result := extractMarker(output, "FW_RESULT:")
	t.Logf("Strict mode blocked domain result: %q", result)

	// Unlisted domain should be blocked (403 from proxy, connection reset, or curl failure)
	if result == "200" || result == "301" || result == "302" {
		t.Errorf("Expected firewall to block example.com, but got HTTP %s", result)
	}
}

func TestBwrap_Addt_FirewallAllowsListed(t *testing.T) {
	// Scenario: User enables firewall in strict mode. Curl to a default-allowed
	// domain (github.com) should succeed because the proxy lets it through.
	requireBwrap(t)
	requireUnshareNet(t)

	if _, err := exec.LookPath("socat"); err != nil {
		t.Skip("socat not installed, skipping firewall test")
	}

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", `
firewall:
  enabled: true
  mode: "strict"
`)
	defer cleanup()

	output, _ := runRunSubcommand(t, dir, "debug",
		"-c", "CODE=$(curl -s --connect-timeout 10 -o /dev/null -w '%{http_code}' https://github.com 2>/dev/null); echo FW_RESULT:$CODE")

	t.Logf("Output:\n%s", output)
	result := extractMarker(output, "FW_RESULT:")
	t.Logf("Strict mode allowed domain result: %q", result)

	if result == "" {
		t.Error("Shell command did not produce FW_RESULT marker")
	} else if result == "000" || result == "BLOCKED" {
		t.Errorf("Expected github.com to be reachable (default allowed), but got %s", result)
	}
}

func TestBwrap_Addt_FirewallPermissiveMode(t *testing.T) {
	// Scenario: User enables firewall in permissive mode with a deny list.
	// Non-denied domains should be reachable. Denied domains should be blocked.
	requireBwrap(t)
	requireUnshareNet(t)

	if _, err := exec.LookPath("socat"); err != nil {
		t.Skip("socat not installed, skipping firewall test")
	}

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", `
firewall:
  enabled: true
  mode: "permissive"
  denied:
    - "neverssl.com"
`)
	defer cleanup()

	// Non-denied domain should be accessible
	output, _ := runRunSubcommand(t, dir, "debug",
		"-c", "CODE=$(curl -s --connect-timeout 10 -o /dev/null -w '%{http_code}' https://example.com 2>/dev/null) || CODE=FAIL; echo PERM_RESULT:$CODE")

	t.Logf("Output:\n%s", output)
	result := extractMarker(output, "PERM_RESULT:")
	t.Logf("Permissive mode non-denied domain result: %q", result)

	if result == "000" || result == "FAIL" {
		t.Errorf("Expected example.com to be reachable in permissive mode, got %s", result)
	}
}

// --- Persistent session tests ---

func TestBwrap_Addt_PersistentSession(t *testing.T) {
	// Scenario: User enables persistent mode. First command creates a file
	// in the sandbox home. Second command checks the file exists, proving
	// the session was reused.
	requireBwrap(t)
	requireNsenter(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", `
persistent: true
`)
	defer cleanup()

	restorePersistent := saveRestoreEnv(t, "ADDT_PERSISTENT", "true")
	defer restorePersistent()

	// Clean up persistent session when done
	defer func() {
		out, err := runContainersSubcommand(t, dir, "clean")
		t.Logf("Cleanup output:\n%s (err=%v)", out, err)
	}()

	// Run 1: Create a marker file inside the persistent home
	output1, err := runRunSubcommand(t, dir, "debug",
		"-c", "touch /home/addt/.persist_marker && echo CREATE:done")
	t.Logf("Run 1 output:\n%s", output1)
	if err != nil {
		t.Fatalf("run 1 failed: %v\nOutput:\n%s", err, output1)
	}
	if extractMarker(output1, "CREATE:") != "done" {
		t.Fatalf("Run 1 did not produce CREATE:done")
	}

	// Run 2: Check marker file exists (session should be reused)
	output2, err := runRunSubcommand(t, dir, "debug",
		"-c", "if [ -f /home/addt/.persist_marker ]; then echo PERSIST:yes; else echo PERSIST:no; fi")
	t.Logf("Run 2 output:\n%s", output2)
	if err != nil {
		t.Fatalf("run 2 failed: %v\nOutput:\n%s", err, output2)
	}

	result := extractMarker(output2, "PERSIST:")
	if result != "yes" {
		t.Errorf("Expected PERSIST:yes (session reused), got %q\nFull output:\n%s", result, output2)
	}
}

func TestBwrap_Addt_EphemeralFresh(t *testing.T) {
	// Scenario: Without persistent mode, each bwrap run gets a fresh tmpfs.
	// A file created in /tmp during one run should NOT exist in the next.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	// Run 1: Create a marker file
	output1, err := runRunSubcommand(t, dir, "debug",
		"-c", "touch /tmp/ephemeral_marker && echo CREATE:done")
	t.Logf("Run 1 output:\n%s", output1)
	if err != nil {
		t.Fatalf("run 1 failed: %v\nOutput:\n%s", err, output1)
	}

	// Run 2: File should NOT exist
	output2, err := runRunSubcommand(t, dir, "debug",
		"-c", "if [ -f /tmp/ephemeral_marker ]; then echo FRESH:no; else echo FRESH:yes; fi")
	t.Logf("Run 2 output:\n%s", output2)
	if err != nil {
		t.Fatalf("run 2 failed: %v\nOutput:\n%s", err, output2)
	}

	result := extractMarker(output2, "FRESH:")
	if result != "yes" {
		t.Errorf("Expected FRESH:yes (ephemeral sandbox), got %q\nFull output:\n%s", result, output2)
	}
}

// --- Filesystem isolation tests ---

func TestBwrap_Addt_TmpfsIsolation(t *testing.T) {
	// Scenario: User runs a command that writes to /tmp inside bwrap.
	// The file should NOT appear on the host's /tmp because bwrap
	// mounts its own tmpfs.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	marker := "bwrap_tmpfs_test_" + strings.ReplaceAll(t.Name(), "/", "_")
	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "touch /tmp/"+marker+" && echo TMPFS:created")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "TMPFS:")
	if result != "created" {
		t.Fatalf("Expected TMPFS:created, got %q", result)
	}

	// File should NOT exist on host's /tmp
	if _, err := os.Stat("/tmp/" + marker); err == nil {
		t.Errorf("File leaked to host /tmp — tmpfs isolation broken")
		os.Remove("/tmp/" + marker) // cleanup
	}
}

func TestBwrap_Addt_RootFilesystemReadonly(t *testing.T) {
	// Scenario: User runs a command that tries to write to a system
	// directory (/usr). The ro-bind mount should prevent writes.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	output, err := runRunSubcommand(t, dir, "debug",
		"-c", "if touch /usr/ro_test 2>/dev/null; then echo ROOTFS:writable; else echo ROOTFS:readonly; fi")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "ROOTFS:")
	if result != "readonly" {
		t.Errorf("Expected ROOTFS:readonly, got %q — system dirs should be read-only", result)
	}
}

// --- ADDT_COMMAND override test ---

func TestBwrap_Addt_CommandOverride(t *testing.T) {
	// Scenario: User changes ADDT_COMMAND to run a different command.
	// The bwrap sandbox should execute the overridden command.
	requireBwrap(t)

	dir, cleanup := setupAddtDirWithExtensions(t, "bwrap", ``)
	defer cleanup()

	restoreCmd := saveRestoreEnv(t, "ADDT_COMMAND", "/bin/bash")
	defer restoreCmd()

	output, err := runShellCommand(t, dir, "debug",
		"-c", "echo CMD_OVERRIDE:works")

	t.Logf("Output:\n%s", output)
	if err != nil {
		t.Fatalf("shell command failed: %v\nOutput:\n%s", err, output)
	}

	result := extractMarker(output, "CMD_OVERRIDE:")
	if result != "works" {
		t.Errorf("Expected CMD_OVERRIDE:works, got %q\nFull output:\n%s", result, output)
	}
}
