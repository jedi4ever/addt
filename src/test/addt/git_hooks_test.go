//go:build addt

package addt

import (
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

// --- Git hooks neutralization tests (in-process config + container tests) ---

func TestGitHooks_Addt_DefaultEnabled(t *testing.T) {
	// Scenario: User starts with no config and checks defaults.
	// git.disable_hooks should default to true (secure by default).
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "git.disable_hooks") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected git.disable_hooks default=true, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected git.disable_hooks source=default, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected output to contain git.disable_hooks, got:\n%s", output)
}

func TestGitHooks_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User disables git hooks neutralization via 'config set git.disable_hooks false'
	// to re-enable pre-commit hooks, then verifies it appears as false in config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "git.disable_hooks", "false"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "git.disable_hooks") {
			if !strings.Contains(line, "false") {
				t.Errorf("Expected git.disable_hooks=false after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected git.disable_hooks source=project after config set, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected output to contain git.disable_hooks, got:\n%s", output)
}

func TestGitHooks_Addt_HooksDisabledInContainer(t *testing.T) {
	// Scenario: User runs a container with default settings (git.disable_hooks=true).
	// Inside the container, 'git config core.hooksPath' should return /dev/null
	// because the git wrapper sets it via GIT_CONFIG_COUNT.
	providers := requireProviders(t)
	cleanupKey := setDummyAnthropicKey(t)
	defer cleanupKey()

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()

			ensureAddtImage(t, dir, "debug")

			output, err := runShellCommand(t, dir, "debug", "-c",
				`echo "HOOKS_PATH:$(git config core.hooksPath 2>/dev/null || echo 'not-set')"`)
			if err != nil {
				t.Fatalf("runShellCommand failed: %v\nOutput: %s", err, output)
			}

			hooksPath := extractMarker(output, "HOOKS_PATH:")
			if hooksPath != "/dev/null" {
				t.Errorf("Expected core.hooksPath=/dev/null, got %q\nFull output:\n%s", hooksPath, output)
			}
		})
	}
}

func TestGitHooks_Addt_ControlEnvVarScrubbed(t *testing.T) {
	// Scenario: User runs a container with git.disable_hooks=true (default).
	// The ADDT_GIT_DISABLE_HOOKS env var should NOT be visible inside the
	// container because the entrypoint unsets it after creating the wrapper.
	providers := requireProviders(t)
	cleanupKey := setDummyAnthropicKey(t)
	defer cleanupKey()

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()

			ensureAddtImage(t, dir, "debug")

			output, err := runShellCommand(t, dir, "debug", "-c",
				`echo "GIT_HOOKS_VAR:${ADDT_GIT_DISABLE_HOOKS:-unset}"`)
			if err != nil {
				t.Fatalf("runShellCommand failed: %v\nOutput: %s", err, output)
			}

			val := extractMarker(output, "GIT_HOOKS_VAR:")
			if val != "unset" {
				t.Errorf("Expected ADDT_GIT_DISABLE_HOOKS to be unset inside container, got %q\nFull output:\n%s", val, output)
			}
		})
	}
}
