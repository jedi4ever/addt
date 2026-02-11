//go:build addt

package addt

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Container tests (subprocess, both providers) ---

func TestShell_Addt_BasicExecution(t *testing.T) {
	// Scenario: User runs `addt shell debug -c "echo SHELL_TEST:hello"`.
	// The shell subcommand should execute the command inside the container
	// and output the echoed value, confirming the shell path works end-to-end.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runShellSubcommand(t, dir, "debug",
				"-c", "echo SHELL_TEST:hello")

			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("shell subcommand failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "SHELL_TEST:")
			if result != "hello" {
				t.Errorf("Expected SHELL_TEST:hello, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestShell_Addt_WorkdirMounted(t *testing.T) {
	// Scenario: User creates a marker file in the project directory, then
	// runs `addt shell debug` to check the file exists at /workspace/.
	// This confirms workdir mounting works in shell mode.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Create a marker file in the project directory
			markerFile := filepath.Join(dir, "shell_test_marker.txt")
			if err := os.WriteFile(markerFile, []byte("MARKER_FOUND"), 0o644); err != nil {
				t.Fatalf("Failed to write marker file: %v", err)
			}

			output, err := runShellSubcommand(t, dir, "debug",
				"-c", "if [ -f /workspace/shell_test_marker.txt ]; then echo WORKDIR_OK:yes; else echo WORKDIR_OK:no; fi")

			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("shell subcommand failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "WORKDIR_OK:")
			if result != "yes" {
				t.Errorf("Expected WORKDIR_OK:yes, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestShell_Addt_BashIsDefault(t *testing.T) {
	// Scenario: User runs `addt shell debug` and checks that
	// ADDT_COMMAND is set to /bin/bash, confirming the shell entrypoint override.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runShellSubcommand(t, dir, "debug",
				"-c", "echo SHELL_CMD:$ADDT_COMMAND")

			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("shell subcommand failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "SHELL_CMD:")
			if result != "/bin/bash" {
				t.Errorf("Expected ADDT_COMMAND=/bin/bash, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestShell_Addt_EnvVarsForwarded(t *testing.T) {
	// Scenario: User sets an env var on the host and configures ADDT_ENV_VARS
	// to forward it. Inside the shell container the var should be available,
	// confirming env forwarding works through the shell subcommand path.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Set the env var on the host and configure forwarding
			origVal := os.Getenv("SHELL_TEST_VAR")
			os.Setenv("SHELL_TEST_VAR", "myvalue")
			defer func() {
				if origVal != "" {
					os.Setenv("SHELL_TEST_VAR", origVal)
				} else {
					os.Unsetenv("SHELL_TEST_VAR")
				}
			}()

			origEnvVars := os.Getenv("ADDT_ENV_VARS")
			os.Setenv("ADDT_ENV_VARS", "SHELL_TEST_VAR")
			defer func() {
				if origEnvVars != "" {
					os.Setenv("ADDT_ENV_VARS", origEnvVars)
				} else {
					os.Unsetenv("ADDT_ENV_VARS")
				}
			}()

			output, err := runShellSubcommand(t, dir, "debug",
				"-c", "echo ENVVAR:${SHELL_TEST_VAR:-NOTSET}")

			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("shell subcommand failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "ENVVAR:")
			if result != "myvalue" {
				t.Errorf("Expected SHELL_TEST_VAR=myvalue, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}

func TestShell_Addt_UserIsAddt(t *testing.T) {
	// Scenario: User runs `whoami` inside the shell container and expects
	// the output to be "addt" (not root), confirming user mapping works.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			if prov == "bwrap" {
				t.Skip("bwrap uses host user, not container-created addt user")
			}
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			output, err := runShellSubcommand(t, dir, "debug",
				"-c", "echo WHOAMI:$(whoami)")

			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("shell subcommand failed: %v\nOutput:\n%s", err, output)
			}

			result := extractMarker(output, "WHOAMI:")
			if result != "addt" {
				t.Errorf("Expected whoami=addt, got %q\nFull output:\n%s", result, output)
			}
		})
	}
}
