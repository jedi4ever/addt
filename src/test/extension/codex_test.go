//go:build extension

package extension

import (
	"os"
	"testing"
)

// Scenario: A user has OPENAI_API_KEY set in their host environment
// and runs the codex extension. The key should be available inside
// the container so that codex can authenticate with the OpenAI API.
func TestCodex_Addt_ApiKeyForwarded(t *testing.T) {
	providers := requireProviders(t)

	const testKey = "sk-test-addt-openai-key-12345"

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// Set up project directory with codex extension config
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			// Set a fake OPENAI_API_KEY so it propagates to the container
			origKey := os.Getenv("OPENAI_API_KEY")
			os.Setenv("OPENAI_API_KEY", testKey)
			defer func() {
				if origKey != "" {
					os.Setenv("OPENAI_API_KEY", origKey)
				} else {
					os.Unsetenv("OPENAI_API_KEY")
				}
			}()

			// Run a shell command inside the container to echo the key
			output, err := runShellCommand(t, dir,
				"codex", "-c", "echo API_KEY_RESULT:$OPENAI_API_KEY")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			// Verify the key arrived inside the container
			result := extractMarker(output, "API_KEY_RESULT:")
			if result != testKey {
				t.Errorf("Expected API_KEY_RESULT:%s, got API_KEY_RESULT:%s\nFull output:\n%s",
					testKey, result, output)
			}
		})
	}
}

// Scenario: With the default security.isolate_secrets: true setting,
// the OPENAI_API_KEY should be delivered through the /run/secrets/
// mechanism rather than as a plain environment variable. This means
// it should NOT appear in /proc/1/environ (the initial process env).
func TestCodex_Addt_ApiKeyNotLeakedToEnv(t *testing.T) {
	providers := requireProviders(t)

	const testKey = "sk-test-addt-openai-key-12345"

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			// Ensure secrets isolation is enabled (the default)
			dir, cleanup := setupAddtDir(t, prov, `
security:
  isolate_secrets: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			// Set a fake OPENAI_API_KEY
			origKey := os.Getenv("OPENAI_API_KEY")
			os.Setenv("OPENAI_API_KEY", testKey)
			defer func() {
				if origKey != "" {
					os.Setenv("OPENAI_API_KEY", origKey)
				} else {
					os.Unsetenv("OPENAI_API_KEY")
				}
			}()

			// Check /proc/1/environ for the key — it should NOT be there
			// when secrets are isolated via tmpfs
			output, err := runShellCommand(t, dir,
				"codex", "-c", procEnvLeakCommand("OPENAI_API_KEY"))
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "PROC_RESULT:")
			if result != "ISOLATED" {
				t.Errorf("Expected OPENAI_API_KEY to be isolated (PROC_RESULT:ISOLATED), got PROC_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user does NOT have OPENAI_API_KEY set in their host
// environment and runs the codex extension. The container should
// not have the variable set either.
func TestCodex_Addt_ApiKeyAbsentWhenUnset(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			// Ensure OPENAI_API_KEY is NOT set in the host environment
			origKey := os.Getenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")
			defer func() {
				if origKey != "" {
					os.Setenv("OPENAI_API_KEY", origKey)
				}
			}()

			// Echo the variable with a fallback to detect absence
			output, err := runShellCommand(t, dir,
				"codex", "-c", "echo API_KEY_RESULT:${OPENAI_API_KEY:-NONE}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "API_KEY_RESULT:")
			if result != "NONE" {
				t.Errorf("Expected API_KEY_RESULT:NONE when OPENAI_API_KEY unset, got API_KEY_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}
