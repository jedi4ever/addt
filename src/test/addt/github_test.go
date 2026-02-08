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

func TestGitHub_Addt_DefaultValues(t *testing.T) {
	// Scenario: User starts with no GitHub config and checks defaults.
	// github.forward_token should default to true, token_source to gh_auth.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	foundToken := false
	foundSource := false
	for _, line := range lines {
		if strings.Contains(line, "github.forward_token") {
			foundToken = true
			if !strings.Contains(line, "true") {
				t.Errorf("Expected github.forward_token default=true, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected github.forward_token source=default, got line: %s", line)
			}
		}
		if strings.Contains(line, "github.token_source") {
			foundSource = true
			if !strings.Contains(line, "gh_auth") {
				t.Errorf("Expected github.token_source default=gh_auth, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected github.token_source source=default, got line: %s", line)
			}
		}
	}
	if !foundToken {
		t.Errorf("Expected output to contain github.forward_token, got:\n%s", output)
	}
	if !foundSource {
		t.Errorf("Expected output to contain github.token_source, got:\n%s", output)
	}
}

func TestGitHub_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User disables GitHub token forwarding via 'config set github.forward_token false',
	// then verifies it appears as false in config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "github.forward_token", "false"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "github.forward_token") {
			if !strings.Contains(line, "false") {
				t.Errorf("Expected github.forward_token=false after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected github.forward_token source=project after config set, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected output to contain github.forward_token, got:\n%s", output)
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

func TestGitHub_Addt_ScopeTokenDefault(t *testing.T) {
	// Scenario: User starts with no GitHub scope config and checks defaults.
	// github.scope_token should default to true, scope_repos should be empty.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	foundScope := false
	foundRepos := false
	for _, line := range lines {
		if strings.Contains(line, "github.scope_token") {
			foundScope = true
			if !strings.Contains(line, "true") {
				t.Errorf("Expected github.scope_token default=true, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected github.scope_token source=default, got line: %s", line)
			}
		}
		if strings.Contains(line, "github.scope_repos") {
			foundRepos = true
			if !strings.Contains(line, "default") {
				t.Errorf("Expected github.scope_repos source=default, got line: %s", line)
			}
		}
	}
	if !foundScope {
		t.Errorf("Expected output to contain github.scope_token, got:\n%s", output)
	}
	if !foundRepos {
		t.Errorf("Expected output to contain github.scope_repos, got:\n%s", output)
	}
}

func TestGitHub_Addt_ScopeTokenConfigSet(t *testing.T) {
	// Scenario: User enables token scoping and sets additional repos via config set,
	// then verifies both appear correctly in config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "github.scope_token", "true"})
	})
	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "github.scope_repos", "org/repo1,org/repo2"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	foundScope := false
	foundRepos := false
	for _, line := range lines {
		if strings.Contains(line, "github.scope_token") {
			foundScope = true
			if !strings.Contains(line, "true") {
				t.Errorf("Expected github.scope_token=true after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected github.scope_token source=project, got line: %s", line)
			}
		}
		if strings.Contains(line, "github.scope_repos") {
			foundRepos = true
			if !strings.Contains(line, "org/repo1,org/repo2") {
				t.Errorf("Expected github.scope_repos=org/repo1,org/repo2, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected github.scope_repos source=project, got line: %s", line)
			}
		}
	}
	if !foundScope {
		t.Errorf("Expected output to contain github.scope_token, got:\n%s", output)
	}
	if !foundRepos {
		t.Errorf("Expected output to contain github.scope_repos, got:\n%s", output)
	}
}

func TestGitHub_Addt_ScopeTokenGHTokenScrubbed(t *testing.T) {
	// Scenario: User enables github.scope_token. Inside the container,
	// GH_TOKEN should be scrubbed from the environment.
	providers := requireProviders(t)
	requireGitHubToken(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
github:
  forward_token: true
  token_source: "env"
  scope_token: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Check that GH_TOKEN env var is not set inside container
			output, err := runShellCommand(t, dir,
				"claude", "-c", "echo ${GH_TOKEN:-NOTSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			if !strings.Contains(output, "NOTSET") {
				t.Errorf("Expected GH_TOKEN to be scrubbed (NOTSET), got:\n%s", output)
			}
		})
	}
}

func TestGitHub_Addt_ScopeTokenCredentialCacheActive(t *testing.T) {
	// Scenario: User enables github.scope_token. Inside the container,
	// git credential.useHttpPath should be set to true.
	providers := requireProviders(t)
	requireGitHubToken(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
github:
  forward_token: true
  token_source: "env"
  scope_token: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Verify git credential.useHttpPath is configured
			output, err := runShellCommand(t, dir,
				"claude", "-c", "git config --global credential.useHttpPath")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			if !strings.Contains(output, "true") {
				t.Errorf("Expected git credential.useHttpPath=true, got:\n%s", output)
			}
		})
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
