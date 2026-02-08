//go:build addt

package addt

import (
	"os"
	"strings"
	"testing"
)

// Scenario: A user enables yolo mode for gemini via project config.
// The env var ADDT_EXTENSION_GEMINI_YOLO should be set inside the container.
func TestGeminiYolo_Addt_ConfigSetsEnvVar(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  gemini:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "gemini")

			output, err := runShellCommand(t, dir,
				"gemini", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_GEMINI_YOLO:-UNSET}")
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

// Scenario: A user does NOT enable yolo mode for gemini. The env var
// should not be set inside the container.
func TestGeminiYolo_Addt_NotSetByDefault(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "gemini")

			output, err := runShellCommand(t, dir,
				"gemini", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_GEMINI_YOLO:-UNSET}")
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

// Scenario: The gemini extension's args.sh passes --yolo through natively
// since gemini CLI supports --yolo directly.
func TestGeminiYolo_Addt_ArgsTransformation(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "gemini")

			output, err := runShellCommand(t, dir,
				"gemini", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/gemini/args.sh --yolo 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--yolo") {
				t.Errorf("Expected args.sh to pass --yolo through, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: When yolo is enabled via config, args.sh should inject
// --yolo even without --yolo on the command line.
func TestGeminiYolo_Addt_ArgsTransformationViaConfig(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
extensions:
  gemini:
    flags:
      yolo: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "gemini")

			output, err := runShellCommand(t, dir,
				"gemini", "-c",
				"echo ARGS_RESULT:$(bash /usr/local/share/addt/extensions/gemini/args.sh 2>/dev/null | tr '\\0' ' ')")
			if err != nil {
				t.Fatalf("shell command failed: %v\nOutput: %s", err, output)
			}

			result := extractMarker(output, "ARGS_RESULT:")
			if !strings.Contains(result, "--yolo") {
				t.Errorf("Expected args.sh to inject --yolo from env var, got ARGS_RESULT:%s\nFull output:\n%s",
					result, output)
			}
		})
	}
}

// Scenario: A user sets yolo via env var for gemini.
func TestGeminiYolo_Addt_EnvVarOverride(t *testing.T) {
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "gemini")

			origVal := os.Getenv("ADDT_EXTENSION_GEMINI_YOLO")
			os.Setenv("ADDT_EXTENSION_GEMINI_YOLO", "true")
			defer func() {
				if origVal != "" {
					os.Setenv("ADDT_EXTENSION_GEMINI_YOLO", origVal)
				} else {
					os.Unsetenv("ADDT_EXTENSION_GEMINI_YOLO")
				}
			}()

			output, err := runShellCommand(t, dir,
				"gemini", "-c", "echo YOLO_RESULT:${ADDT_EXTENSION_GEMINI_YOLO:-UNSET}")
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
