//go:build addt

package addt

import (
	"os"
	"strings"
	"testing"
)

// --- Sandbox isolation tests (multi-provider) ---

func TestSandbox_Addt_EnvIsolation(t *testing.T) {
	// Scenario: User runs a command without forwarding a host env var.
	// The container should NOT leak host environment variables.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			restoreVar := saveRestoreEnv(t, "SANDBOX_LEAK_TEST", "should_not_leak")
			defer restoreVar()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo LEAK:${SANDBOX_LEAK_TEST:-CLEAN}")

			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "LEAK:")
			if result != "CLEAN" {
				t.Errorf("Expected LEAK:CLEAN (env should not leak), got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestSandbox_Addt_PidIsolation(t *testing.T) {
	// Scenario: User runs a command that checks PID 1 inside the container.
	// PID 1 should not be the host's init process, confirming PID namespace
	// isolation.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo PID1:$(cat /proc/1/comm 2>/dev/null || echo unknown)")

			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run subcommand failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "PID1:")
			// In an isolated container, PID 1 should NOT be the host's init/systemd
			if result == "systemd" || result == "init" {
				t.Errorf("Expected PID isolation (PID 1 != host init), got %q", result)
			}
		})
	}
}

func TestSandbox_Addt_TmpfsIsolation(t *testing.T) {
	// Scenario: User writes to /tmp inside the container. The file should
	// NOT appear on the host's /tmp, confirming filesystem isolation.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			marker := "sandbox_tmpfs_" + strings.ReplaceAll(t.Name(), "/", "_")
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
				t.Errorf("File leaked to host /tmp — filesystem isolation broken")
				os.Remove("/tmp/" + marker)
			}
		})
	}
}

func TestSandbox_Addt_NetworkIsolation(t *testing.T) {
	// Scenario: User enables network isolation (security.network_mode: none).
	// Network should be completely unavailable inside the container.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			if prov == "bwrap" {
				requireUnshareNet(t)
			}

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
security:
  network_mode: "none"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, _ := runRunSubcommand(t, dir, "debug",
				"-c", "if curl -s --connect-timeout 2 http://example.com >/dev/null 2>&1; then echo NET:open; else echo NET:isolated; fi")

			t.Logf("Output:\n%s", output)

			result := extractMarker(output, "NET:")
			if result != "isolated" {
				t.Errorf("Expected NET:isolated (no network), got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestSandbox_Addt_WorkdirReadonly(t *testing.T) {
	// Scenario: User enables read-only workdir. Writing to /workspace/
	// inside the container should fail.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
workdir:
  readonly: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

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
		})
	}
}
