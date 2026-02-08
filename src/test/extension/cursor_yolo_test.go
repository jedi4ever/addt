//go:build extension

package extension

import (
	"os"
	"strings"
	"testing"
)

// Scenario: A user enables yolo mode for cursor via project config.
// The env var ADDT_EXTENSION_CURSOR_YOLO should be set inside the container.
func TestCursorYolo_Addt_ConfigSetsEnvVar(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  cursor:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "cursor")

			output, err := runShellCommand(t, dir,
				"cursor", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CURSOR_YOLO:-UNSET}")
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

// Scenario: A user does NOT enable yolo mode for cursor. The env var
// should not be set inside the container.
func TestCursorYolo_Addt_NotSetByDefault(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "cursor")

			output, err := runShellCommand(t, dir,
				"cursor", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CURSOR_YOLO:-UNSET}")
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

// Scenario: The cursor extension's args.sh transforms --yolo into
// --force so that cursor runs in force mode.
func TestCursorYolo_Addt_ArgsTransformation(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "cursor")

			output, err := runShellCommand(t, dir,
				"cursor", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/cursor/args.sh --yolo 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--force") {
				t.Errorf("Expected args.sh to transform --yolo to --force, got ARGS_RESULT:%s\nFull output:\n%s",
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
// --force even without --yolo on the command line.
func TestCursorYolo_Addt_ArgsTransformationViaConfig(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  cursor:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "cursor")

			output, err := runShellCommand(t, dir,
				"cursor", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/cursor/args.sh 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--force") {
				t.Errorf("Expected args.sh to inject --force from env var, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user sets yolo via env var for cursor.
func TestCursorYolo_Addt_EnvVarOverride(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "cursor")

			origVal := os.Getenv("ADDT_EXTENSION_CURSOR_YOLO")
			os.Setenv("ADDT_EXTENSION_CURSOR_YOLO", "true")
			defer func() {
				if origVal != "" {
					os.Setenv("ADDT_EXTENSION_CURSOR_YOLO", origVal)
				} else {
					os.Unsetenv("ADDT_EXTENSION_CURSOR_YOLO")
				}
			}()

			output, err := runShellCommand(t, dir,
				"cursor", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_CURSOR_YOLO:-UNSET}")
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
