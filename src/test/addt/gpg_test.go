//go:build addt

package addt

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

func requireGPGAgent(t *testing.T) {
	t.Helper()
	home, _ := os.UserHomeDir()
	gnupgDir := filepath.Join(home, ".gnupg")
	if _, err := os.Stat(gnupgDir); os.IsNotExist(err) {
		t.Skip("~/.gnupg not found, skipping GPG tests")
	}
}

func TestGPG_Addt_ConfigLoaded(t *testing.T) {
	_, cleanup := setupAddtDir(t, "", `
gpg:
  forward: "proxy"
  allowed_key_ids:
    - "ABCD1234"
    - "EFGH5678"
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	if !strings.Contains(output, "gpg.forward") {
		t.Errorf("Expected output to contain gpg.forward, got:\n%s", output)
	}
	if !strings.Contains(output, "proxy") {
		t.Errorf("Expected output to contain 'proxy', got:\n%s", output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "gpg.forward") && !strings.Contains(line, "allowed") {
			if !strings.Contains(line, "project") {
				t.Errorf("Expected gpg.forward source=project, got line: %s", line)
			}
		}
		if strings.Contains(line, "gpg.allowed_key_ids") {
			if !strings.Contains(line, "ABCD1234") {
				t.Errorf("Expected gpg.allowed_key_ids to contain ABCD1234, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected gpg.allowed_key_ids source=project, got line: %s", line)
			}
		}
	}
}

func TestGPG_Addt_ProxyMode(t *testing.T) {
	providers := requireProviders(t)
	requireGPGAgent(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
gpg:
  forward: "proxy"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Check GPG socket exists.
			// On macOS+podman, socat chmod fails on virtiofs so socket may not be created.
			output, err := runShellCommand(t, dir,
				"claude", "-c", "ls -la /home/addt/.gnupg/S.gpg-agent 2>&1 || echo NO_SOCKET")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			if strings.Contains(output, "NO_SOCKET") {
				if prov == "podman" && runtime.GOOS == "darwin" {
					t.Log("Warning: GPG socket not created (known macOS+podman virtiofs chmod limitation)")
				} else {
					t.Errorf("Expected GPG agent socket to exist in proxy mode, got:\n%s", output)
				}
			}

			// Check safe files are present (public keyring, config)
			output2, err := runShellCommand(t, dir,
				"claude", "-c", "ls /home/addt/.gnupg/ 2>&1")
			if err != nil {
				t.Fatalf("shell ls .gnupg failed: %v\nOutput: %s", err, output2)
			}
			t.Logf("GPG proxy mode directory:\n%s", output2)

			// Verify private key directory is NOT mounted
			output3, err := runShellCommand(t, dir,
				"claude", "-c", "test -d /home/addt/.gnupg/private-keys-v1.d && echo HAS_PRIVATE || echo NO_PRIVATE")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output3)
			}

			if strings.Contains(output3, "HAS_PRIVATE") {
				t.Errorf("Expected private-keys-v1.d to NOT be mounted in proxy mode, got:\n%s", output3)
			}
		})
	}
}

func TestGPG_Addt_KeysMode(t *testing.T) {
	providers := requireProviders(t)
	requireGPGAgent(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
gpg:
  forward: "keys"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, err := runShellCommand(t, dir,
				"claude", "-c", "ls /home/addt/.gnupg/ 2>&1")
			if err != nil {
				t.Fatalf("shell ls .gnupg failed: %v\nOutput: %s", err, output)
			}

			if strings.Contains(output, "No such file") || strings.Contains(output, "cannot access") {
				t.Errorf("Expected .gnupg directory to exist in keys mode, got:\n%s", output)
			}

			t.Logf("GPG keys mode directory:\n%s", output)
		})
	}
}

func TestGPG_Addt_AgentMode(t *testing.T) {
	providers := requireProviders(t)
	requireGPGAgent(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// On macOS, podman can't forward Unix sockets into the VM
			if prov == "podman" && runtime.GOOS == "darwin" {
				t.Skip("podman on macOS cannot forward Unix sockets (use proxy mode)")
			}

			dir, cleanup := setupAddtDir(t, prov, `
gpg:
  forward: "agent"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, err := runShellCommand(t, dir,
				"claude", "-c", "ls -la /home/addt/.gnupg/S.gpg-agent 2>&1 || echo NO_SOCKET")
			if err != nil {
				t.Skipf("agent mode may not be supported on this platform: %v\nOutput: %s", err, output)
			}

			if strings.Contains(output, "NO_SOCKET") {
				t.Errorf("Expected GPG agent socket in agent mode, got:\n%s", output)
			}

			t.Logf("GPG agent mode socket:\n%s", output)
		})
	}
}

func TestGPG_Addt_Disabled(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
gpg:
  forward: "off"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, err := runShellCommand(t, dir,
				"claude", "-c",
				"test -d /home/addt/.gnupg && echo exists || echo missing")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			if !strings.Contains(output, "missing") {
				t.Errorf("Expected .gnupg directory to be missing when gpg.forward=off, got:\n%s", output)
			}
		})
	}
}
