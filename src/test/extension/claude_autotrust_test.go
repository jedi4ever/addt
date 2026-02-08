//go:build extension

package extension

import (
	"testing"
)

// Scenario: When autotrust is enabled for claude, the setup.sh should
// create the workspace trust entry in ~/.claude.json inside the container.
func TestWorkdir_Addt_ClaudeWorkspaceTrusted(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			defer setDummyAnthropicKey(t)()

			dir, cleanup := setupAddtDir(t, prov, `
workdir:
  autotrust: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Check that ~/.claude.json has the workspace trust entry
			output, err := runShellCommand(t, dir,
				"claude", "-c",
				"cat ~/.claude.json | grep -o '/workspace' | head -1 | xargs -I{} echo TRUST_RESULT:{}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "TRUST_RESULT:")
			if result != "/workspace" {
				t.Errorf("Expected /workspace trust entry in ~/.claude.json, got TRUST_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: When autotrust is disabled for claude, the setup.sh should
// NOT create the workspace trust entry in ~/.claude.json.
func TestWorkdir_Addt_ClaudeWorkspaceNotTrusted(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			defer setDummyAnthropicKey(t)()

			dir, cleanup := setupAddtDir(t, prov, `
workdir:
  autotrust: false
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Check that ~/.claude.json does NOT have the workspace trust entry
			output, err := runShellCommand(t, dir,
				"claude", "-c",
				"if grep -q '/workspace' ~/.claude.json 2>/dev/null; then echo TRUST_RESULT:TRUSTED; else echo TRUST_RESULT:NOT_TRUSTED; fi")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "TRUST_RESULT:")
			if result != "NOT_TRUSTED" {
				t.Errorf("Expected workspace NOT trusted when autotrust=false, got TRUST_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}
