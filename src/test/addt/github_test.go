//go:build addt

package addt

import (
	"os"
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

func requireGitHubToken(t *testing.T) {
	t.Helper()
	if os.Getenv("GH_TOKEN") != "" {
		return
	}
	// Try gh CLI as fallback
	if token := runCmd(t, "gh", "auth", "token"); token != "" {
		os.Setenv("GH_TOKEN", token)
		return
	}
	t.Skip("No GitHub token available (GH_TOKEN not set and gh auth token failed)")
}

func TestGitHub_Addt_ConfigLoaded(t *testing.T) {
	_, cleanup := setupAddtDir(t, "", `
github:
  forward_token: true
  token_source: "env"
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	if !strings.Contains(output, "github.forward_token") {
		t.Errorf("Expected output to contain github.forward_token, got:\n%s", output)
	}
	if !strings.Contains(output, "github.token_source") {
		t.Errorf("Expected output to contain github.token_source, got:\n%s", output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "github.forward_token") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected github.forward_token=true, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected github.forward_token source=project, got line: %s", line)
			}
		}
		if strings.Contains(line, "github.token_source") {
			if !strings.Contains(line, "env") {
				t.Errorf("Expected github.token_source=env, got line: %s", line)
			}
		}
	}
}

func TestGitHub_Addt_TokenForwarded(t *testing.T) {
	providers := requireProviders(t)
	requireGitHubToken(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
github:
  forward_token: true
  token_source: "env"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// gh auth status checks if the token works
			output, err := runShellCommand(t, dir,
				"claude", "-c", "gh auth status")
			if err != nil {
				t.Fatalf("gh auth status failed: %v\nOutput: %s", err, output)
			}

			outputLower := strings.ToLower(output)
			if !strings.Contains(outputLower, "logged in") {
				t.Errorf("Expected 'logged in' in gh auth status output, got:\n%s", output)
			}
		})
	}
}

func TestGitHub_Addt_TokenDisabled(t *testing.T) {
	providers := requireProviders(t)
	requireGitHubToken(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
github:
  forward_token: false
  token_source: "env"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// With token forwarding disabled, gh auth status should fail
			output, err := runShellCommand(t, dir,
				"claude", "-c", "gh auth status 2>&1 || true")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			// Should NOT be logged in.
			// "Logged in to" is the positive auth message from gh auth status.
			// "not logged into" is the negative message - don't match on just "logged in".
			if strings.Contains(output, "Logged in to") {
				t.Errorf("Expected gh auth to NOT be logged in when forward_token=false, got:\n%s", output)
			}
		})
	}
}
