//go:build extension

package extension

import (
	"strings"
	"testing"
)

// Scenario: A user enables yolo mode via project config so that
// claude receives --dangerously-skip-permissions. The env var
// ADDT_EXTENSION_CLAUDE_YOLO should be set inside the container.
func TestClaudeYolo_Addt_ConfigSetsEnvVar(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			defer setDummyAnthropicKey(t)()

			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  claude:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, err := runShellCommand(t, dir,
				"claude", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CLAUDE_YOLO:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "YOLO_RESULT:")
			if result != "true" {
				t.Errorf("Expected YOLO_RESULT:true, got YOLO_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user does NOT enable yolo mode. The env var
// ADDT_EXTENSION_CLAUDE_YOLO should not be set inside the container.
func TestClaudeYolo_Addt_NotSetByDefault(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			defer setDummyAnthropicKey(t)()

			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, err := runShellCommand(t, dir,
				"claude", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CLAUDE_YOLO:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "YOLO_RESULT:")
			if result != "UNSET" {
				t.Errorf("Expected YOLO_RESULT:UNSET when yolo not configured, got YOLO_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: Inside the container, the claude extension's args.sh script
// transforms --yolo into --dangerously-skip-permissions so that claude
// receives the correct flag. Verify the transformation works.
func TestClaudeYolo_Addt_ArgsTransformation(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			defer setDummyAnthropicKey(t)()

			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, err := runShellCommand(t, dir,
				"claude", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/claude/args.sh --yolo 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--dangerously-skip-permissions") {
				t.Errorf("Expected args.sh to transform --yolo to --dangerously-skip-permissions, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
			if strings.Contains(result, "--yolo") {
				t.Errorf("Expected --yolo to be removed after transformation, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: When yolo is enabled via config (env var), args.sh should
// inject --dangerously-skip-permissions even without --yolo on the command
// line. This is the config-driven path.
func TestClaudeYolo_Addt_ArgsTransformationViaEnv(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			defer setDummyAnthropicKey(t)()

			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  claude:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, err := runShellCommand(t, dir,
				"claude", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/claude/args.sh 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--dangerously-skip-permissions") {
				t.Errorf("Expected args.sh to inject --dangerously-skip-permissions from env var, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user sets yolo via the ADDT_EXTENSION_CLAUDE_YOLO env var
// directly, without any config file setting. The env var should propagate
// to the container and args.sh should pick it up.
func TestClaudeYolo_Addt_EnvVarOverride(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			defer setDummyAnthropicKey(t)()

			// No yolo in config
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			// Set yolo via env var
			t.Setenv("ADDT_EXTENSION_CLAUDE_YOLO", "true")

			output, err := runShellCommand(t, dir,
				"claude", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CLAUDE_YOLO:-UNSET}")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "YOLO_RESULT:")
			if result != "true" {
				t.Errorf("Expected YOLO_RESULT:true (env var override), got YOLO_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}
