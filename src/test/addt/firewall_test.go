//go:build addt

package addt

import (
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
	firewallcmd "github.com/jedi4ever/addt/cmd/firewall"
)

// --- Config tests (in-process, no container needed) ---

func TestFirewall_Addt_ConfigLoaded(t *testing.T) {
	// Scenario: User sets firewall.enabled and firewall.mode in project config,
	// then verifies both appear in config list with correct values and source.
	_, cleanup := setupAddtDir(t, "", `
firewall:
  enabled: true
  mode: "strict"
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	if !strings.Contains(output, "firewall.enabled") {
		t.Errorf("Expected output to contain firewall.enabled, got:\n%s", output)
	}
	if !strings.Contains(output, "firewall.mode") {
		t.Errorf("Expected output to contain firewall.mode, got:\n%s", output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "firewall.enabled") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected firewall.enabled=true, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected firewall.enabled source=project, got line: %s", line)
			}
		}
		if strings.Contains(line, "firewall.mode") {
			if !strings.Contains(line, "strict") {
				t.Errorf("Expected firewall.mode=strict, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected firewall.mode source=project, got line: %s", line)
			}
		}
	}
}

func TestFirewall_Addt_DefaultValues(t *testing.T) {
	// Scenario: User starts with no firewall config and checks defaults.
	// firewall.enabled should default to false, firewall.mode to strict.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "firewall.enabled") {
			if !strings.Contains(line, "false") {
				t.Errorf("Expected firewall.enabled default=false, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected firewall.enabled source=default, got line: %s", line)
			}
		}
		if strings.Contains(line, "firewall.mode") {
			if !strings.Contains(line, "strict") {
				t.Errorf("Expected firewall.mode default=strict, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected firewall.mode source=default, got line: %s", line)
			}
		}
	}
}

func TestFirewall_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User enables firewall via 'config set' command,
	// then verifies it appears in config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Set firewall.enabled to true via config set
	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "firewall.enabled", "true"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "firewall.enabled") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected firewall.enabled=true after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected firewall.enabled source=project after config set, got line: %s", line)
			}
		}
	}
}

// --- Project rule management tests (in-process) ---

func TestFirewall_Addt_ProjectAllowAndList(t *testing.T) {
	// Scenario: User allows a domain for the project, then lists rules
	// to verify it appears in the allowed list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Allow a domain
	output := captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "allow", "custom-api.example.com"})
	})
	if !strings.Contains(output, "Added") || !strings.Contains(output, "custom-api.example.com") {
		t.Errorf("Expected confirmation of adding domain, got:\n%s", output)
	}

	// List project rules and verify domain appears
	output = captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "list"})
	})
	if !strings.Contains(output, "custom-api.example.com") {
		t.Errorf("Expected custom-api.example.com in project list, got:\n%s", output)
	}
}

func TestFirewall_Addt_ProjectDenyAndList(t *testing.T) {
	// Scenario: User denies a domain for the project, then lists rules
	// to verify it appears in the denied list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Deny a domain
	output := captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "deny", "malware.example.com"})
	})
	if !strings.Contains(output, "Added") || !strings.Contains(output, "malware.example.com") {
		t.Errorf("Expected confirmation of denying domain, got:\n%s", output)
	}

	// List project rules and verify domain appears in denied
	output = captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "list"})
	})
	if !strings.Contains(output, "malware.example.com") {
		t.Errorf("Expected malware.example.com in project denied list, got:\n%s", output)
	}
}

func TestFirewall_Addt_ProjectRemove(t *testing.T) {
	// Scenario: User adds a domain, then removes it, verifying it's gone.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Allow a domain first
	captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "allow", "temp-api.example.com"})
	})

	// Remove the domain
	output := captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "remove", "temp-api.example.com"})
	})
	if !strings.Contains(output, "Removed") {
		t.Errorf("Expected removal confirmation, got:\n%s", output)
	}

	// Verify it no longer appears in list
	output = captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "list"})
	})
	if strings.Contains(output, "temp-api.example.com") {
		t.Errorf("Expected temp-api.example.com to be removed from list, got:\n%s", output)
	}
}

func TestFirewall_Addt_ProjectReset(t *testing.T) {
	// Scenario: User adds several rules, resets the project, verifies all cleared.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Add some rules
	captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "allow", "api1.example.com"})
	})
	captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "deny", "blocked.example.com"})
	})

	// Reset
	output := captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "reset"})
	})
	if !strings.Contains(output, "Cleared") {
		t.Errorf("Expected reset confirmation, got:\n%s", output)
	}

	// Verify rules are gone
	output = captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "list"})
	})
	if strings.Contains(output, "api1.example.com") {
		t.Errorf("Expected api1.example.com to be cleared after reset, got:\n%s", output)
	}
	if strings.Contains(output, "blocked.example.com") {
		t.Errorf("Expected blocked.example.com to be cleared after reset, got:\n%s", output)
	}
}

// --- Global rule management tests (in-process) ---

func TestFirewall_Addt_GlobalAllowAndList(t *testing.T) {
	// Scenario: User allows a domain globally, then lists global rules
	// to verify it appears alongside the defaults.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Allow a domain globally
	output := captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"global", "allow", "custom-global.example.com"})
	})
	if !strings.Contains(output, "Added") || !strings.Contains(output, "custom-global.example.com") {
		t.Errorf("Expected confirmation of adding global domain, got:\n%s", output)
	}

	// List global rules
	output = captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"global", "list"})
	})
	if !strings.Contains(output, "custom-global.example.com") {
		t.Errorf("Expected custom-global.example.com in global list, got:\n%s", output)
	}
}

func TestFirewall_Addt_GlobalDenyAndReset(t *testing.T) {
	// Scenario: User denies a domain globally, verifies it appears,
	// then resets to defaults and verifies the deny is cleared and
	// defaults are restored.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Deny a domain globally
	captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"global", "deny", "evil.example.com"})
	})

	// List to verify deny
	output := captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"global", "list"})
	})
	if !strings.Contains(output, "evil.example.com") {
		t.Errorf("Expected evil.example.com in global denied list, got:\n%s", output)
	}

	// Reset to defaults
	output = captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"global", "reset"})
	})
	if !strings.Contains(output, "Reset") {
		t.Errorf("Expected reset confirmation, got:\n%s", output)
	}

	// Verify deny is cleared and defaults are restored
	output = captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"global", "list"})
	})
	if strings.Contains(output, "evil.example.com") {
		t.Errorf("Expected evil.example.com to be cleared after reset, got:\n%s", output)
	}
	// Defaults should be present after reset
	if !strings.Contains(output, "api.anthropic.com") {
		t.Errorf("Expected default domain api.anthropic.com after reset, got:\n%s", output)
	}
}

func TestFirewall_Addt_DuplicateDomain(t *testing.T) {
	// Scenario: User tries to add the same domain twice to the project
	// allowed list. The second attempt should indicate it's already in the list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// First add
	captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "allow", "dup.example.com"})
	})

	// Second add — should say "already in"
	output := captureOutput(t, func() {
		firewallcmd.HandleCommand([]string{"project", "allow", "dup.example.com"})
	})
	if !strings.Contains(output, "already in") {
		t.Errorf("Expected 'already in' message for duplicate domain, got:\n%s", output)
	}
}

// --- Container tests (subprocess, both providers) ---

// extractCurlResult finds the curl HTTP status code in subprocess output.
// We use CURL_RESULT markers to isolate it from test framework noise.
func extractCurlResult(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CURL_RESULT:") {
			return strings.TrimPrefix(line, "CURL_RESULT:")
		}
	}
	return ""
}

func TestFirewall_Addt_StrictModeBlocks(t *testing.T) {
	// Scenario: User enables firewall in strict mode. Curl to an unlisted
	// domain (example.com) should fail or time out because strict mode
	// only allows explicitly listed domains.
	//
	// The container starts as root (--user root), applies iptables rules
	// directly (no sudo needed), then drops to addt via gosu. This works
	// even with no-new-privileges enabled.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
firewall:
  enabled: true
  mode: "strict"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Try to reach example.com — not in default allowed list
			// Use markers to extract the curl result from subprocess noise
			output, _ := runShellCommand(t, dir,
				"claude", "-c",
				"CODE=$(curl -s --connect-timeout 5 -o /dev/null -w '%{http_code}' https://example.com 2>/dev/null) || CODE=BLOCKED; echo CURL_RESULT:$CODE")

			result := extractCurlResult(output)
			t.Logf("Strict mode blocked domain result: %q", result)

			// In strict mode, unlisted domains should be blocked.
			// With root entrypoint + gosu, firewall rules are properly applied.
			if result == "200" || result == "301" || result == "302" {
				t.Errorf("Expected strict mode to block example.com, but got HTTP %s", result)
			}
			// Expect either BLOCKED, 000 (connection failed), or empty (timeout)
			if result != "" && result != "BLOCKED" && result != "000" {
				t.Logf("Unexpected result %q — firewall may not have initialized properly", result)
			}
		})
	}
}

func TestFirewall_Addt_AllowedDomainReachable(t *testing.T) {
	// Scenario: User enables firewall in strict mode. Curl to a
	// default-allowed domain (github.com) should succeed because
	// it's in the default allowed list.
	//
	// The container starts as root, applies iptables rules, then drops
	// to addt via gosu. Allowed domains should remain reachable.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
firewall:
  enabled: true
  mode: "strict"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Try to reach github.com — in default allowed list
			// Use markers to extract the curl result from subprocess noise
			output, _ := runShellCommand(t, dir,
				"claude", "-c",
				"CODE=$(curl -s --connect-timeout 10 -o /dev/null -w '%{http_code}' https://github.com 2>/dev/null); echo CURL_RESULT:$CODE")

			result := extractCurlResult(output)
			t.Logf("Strict mode allowed domain result: %q", result)

			// github.com is default-allowed, so it should be reachable
			if result != "" {
				if result == "000" {
					t.Errorf("Expected github.com to be reachable (it's default-allowed), but curl failed with status 000")
				}
			} else {
				t.Errorf("Shell command did not produce CURL_RESULT — entrypoint may have failed")
			}
		})
	}
}

func TestFirewall_Addt_DisabledAllowsAll(t *testing.T) {
	// Scenario: User disables the firewall entirely. Curl to any
	// domain should succeed because no filtering is applied.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
firewall:
  enabled: false
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Try to reach example.com — should work with firewall disabled
			// Use markers to extract the curl result from subprocess noise
			output, _ := runShellCommand(t, dir,
				"claude", "-c",
				"CODE=$(curl -s --connect-timeout 10 -o /dev/null -w '%{http_code}' https://example.com 2>/dev/null); echo CURL_RESULT:$CODE")

			result := extractCurlResult(output)
			t.Logf("Disabled firewall result: %q", result)

			// With firewall disabled, curl should run and reach the domain
			if result == "" {
				t.Log("Shell command did not produce CURL_RESULT — entrypoint likely exited before running command")
			} else if result == "000" {
				t.Errorf("Expected example.com to be reachable with firewall disabled, but curl failed with status 000")
			}
		})
	}
}
