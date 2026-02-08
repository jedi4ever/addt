//go:build addt

package addt

import (
	"os"
	"strings"
	"testing"
)

// Scenario: A user enables yolo mode for codex via project config.
// The env var ADDT_EXTENSION_CODEX_YOLO should be set inside the container.
func TestCodexYolo_Addt_ConfigSetsEnvVar(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  codex:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			output, err := runShellCommand(t, dir,
				"codex", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CODEX_YOLO:-UNSET}")
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

// Scenario: A user does NOT enable yolo mode for codex. The env var
// should not be set inside the container.
func TestCodexYolo_Addt_NotSetByDefault(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			output, err := runShellCommand(t, dir,
				"codex", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CODEX_YOLO:-UNSET}")
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

// Scenario: The codex extension's args.sh transforms --yolo into
// --full-auto so that codex runs in full autonomous mode.
func TestCodexYolo_Addt_ArgsTransformation(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			output, err := runShellCommand(t, dir,
				"codex", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/codex/args.sh --yolo 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--full-auto") {
				t.Errorf("Expected args.sh to transform --yolo to --full-auto, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
			if strings.Contains(result, "--yolo") {
				t.Errorf("Expected --yolo to be removed after transformation, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: When yolo is enabled via config, args.sh should inject
// --full-auto even without --yolo on the command line.
func TestCodexYolo_Addt_ArgsTransformationViaConfig(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  codex:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			output, err := runShellCommand(t, dir,
				"codex", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/codex/args.sh 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--full-auto") {
				t.Errorf("Expected args.sh to inject --full-auto from env var, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user sets yolo via env var for codex.
func TestCodexYolo_Addt_EnvVarOverride(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "codex")

			origVal := os.Getenv("ADDT_EXTENSION_CODEX_YOLO")
			os.Setenv("ADDT_EXTENSION_CODEX_YOLO", "true")
			defer func() {
				if origVal != "" {
					os.Setenv("ADDT_EXTENSION_CODEX_YOLO", origVal)
				} else {
					os.Unsetenv("ADDT_EXTENSION_CODEX_YOLO")
				}
			}()

			output, err := runShellCommand(t, dir,
				"codex", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CODEX_YOLO:-UNSET}")
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
