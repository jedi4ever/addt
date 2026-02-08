//go:build addt

package addt

import (
	"os"
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

// --- Config tests (in-process, no container needed) ---

func TestOtel_Addt_DefaultValues(t *testing.T) {
	// Scenario: User starts with no OTEL config. The defaults should be
	// otel.enabled=false, otel.endpoint=http://host.docker.internal:4318,
	// otel.protocol=http/json, otel.service_name=addt, otel.headers="" —
	// all with source=default.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")

	checks := map[string]string{
		"otel.enabled":      "false",
		"otel.endpoint":     "http://host.docker.internal:4318",
		"otel.protocol":     "http/json",
		"otel.service_name": "addt",
	}

	for key, expectedValue := range checks {
		found := false
		for _, line := range lines {
			if strings.Contains(line, key) {
				found = true
				if !strings.Contains(line, expectedValue) {
					t.Errorf("Expected %s default=%s, got line: %s", key, expectedValue, line)
				}
				if !strings.Contains(line, "default") {
					t.Errorf("Expected %s source=default, got line: %s", key, line)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected %s key in config list, got:\n%s", key, output)
		}
	}
}

func TestOtel_Addt_ConfigLoaded(t *testing.T) {
	// Scenario: User sets all OTEL keys in .addt.yaml project config.
	// Verify they appear in config list with correct values and source=project.
	_, cleanup := setupAddtDir(t, "", `
otel:
  enabled: true
  endpoint: http://localhost:4317
  protocol: grpc
  service_name: my-service
  headers: "Authorization=Bearer token123"
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")

	checks := map[string]string{
		"otel.enabled":      "true",
		"otel.endpoint":     "http://localhost:4317",
		"otel.protocol":     "grpc",
		"otel.service_name": "my-service",
		"otel.headers":      "Authorization=Bearer token123",
	}

	for key, expectedValue := range checks {
		found := false
		for _, line := range lines {
			if strings.Contains(line, key) {
				found = true
				if !strings.Contains(line, expectedValue) {
					t.Errorf("Expected %s=%s, got line: %s", key, expectedValue, line)
				}
				if !strings.Contains(line, "project") {
					t.Errorf("Expected %s source=project, got line: %s", key, line)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected %s key in config list, got:\n%s", key, output)
		}
	}
}

func TestOtel_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User enables OTEL via 'config set' command, then verifies
	// it appears in config list with value=true and source=project.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "otel.enabled", "true"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "otel.enabled") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected otel.enabled=true after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected otel.enabled source=project after config set, got line: %s", line)
			}
			return
		}
	}
	t.Errorf("Expected otel.enabled key in config list, got:\n%s", output)
}

// --- Container tests (subprocess, both providers) ---

func TestOtel_Addt_EnvVarsInjected(t *testing.T) {
	// Scenario: User enables OTEL in config. When running a container,
	// the standard OTEL env vars should be injected into the container
	// environment (OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_SERVICE_NAME, etc.).
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
otel:
  enabled: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			defer saveRestoreEnv(t, "ADDT_OTEL_ENABLED", "true")()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo ENDPOINT:${OTEL_EXPORTER_OTLP_ENDPOINT:-NOTSET} && echo SVCNAME:${OTEL_SERVICE_NAME:-NOTSET} && echo PROTOCOL:${OTEL_EXPORTER_OTLP_PROTOCOL:-NOTSET} && echo TELEMETRY:${CLAUDE_CODE_ENABLE_TELEMETRY:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			endpoint := extractMarker(output, "ENDPOINT:")
			if endpoint == "NOTSET" || endpoint == "" {
				t.Errorf("Expected OTEL_EXPORTER_OTLP_ENDPOINT to be set, got %q", endpoint)
			}

			svcName := extractMarker(output, "SVCNAME:")
			if svcName == "NOTSET" || svcName == "" {
				t.Errorf("Expected OTEL_SERVICE_NAME to be set, got %q", svcName)
			}

			protocol := extractMarker(output, "PROTOCOL:")
			if protocol == "NOTSET" || protocol == "" {
				t.Errorf("Expected OTEL_EXPORTER_OTLP_PROTOCOL to be set, got %q", protocol)
			}

			telemetry := extractMarker(output, "TELEMETRY:")
			if telemetry != "1" {
				t.Errorf("Expected CLAUDE_CODE_ENABLE_TELEMETRY=1, got %q", telemetry)
			}
		})
	}
}

func TestOtel_Addt_ServiceNameWithExtension(t *testing.T) {
	// Scenario: User enables OTEL with default service_name "addt" and runs
	// the "debug" extension. The OTEL_SERVICE_NAME should become "addt-debug"
	// (extension name appended to default service name).
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
otel:
  enabled: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			defer saveRestoreEnv(t, "ADDT_OTEL_ENABLED", "true")()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo SVCNAME:${OTEL_SERVICE_NAME:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			svcName := extractMarker(output, "SVCNAME:")
			if svcName != "addt-debug" {
				t.Errorf("Expected OTEL_SERVICE_NAME=addt-debug, got %q", svcName)
			}
		})
	}
}

func TestOtel_Addt_CustomServiceName(t *testing.T) {
	// Scenario: User sets a custom service_name. It should be preserved
	// as-is — the extension name should NOT be appended.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
otel:
  enabled: true
  service_name: my-custom-service
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			defer saveRestoreEnv(t, "ADDT_OTEL_ENABLED", "true")()
			defer saveRestoreEnv(t, "ADDT_OTEL_SERVICE_NAME", "my-custom-service")()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo SVCNAME:${OTEL_SERVICE_NAME:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			svcName := extractMarker(output, "SVCNAME:")
			if svcName != "my-custom-service" {
				t.Errorf("Expected OTEL_SERVICE_NAME=my-custom-service, got %q", svcName)
			}
		})
	}
}

func TestOtel_Addt_DisabledNoVars(t *testing.T) {
	// Scenario: User does NOT enable OTEL (default). The OTEL env vars
	// should NOT be set inside the container.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, ``)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			// Ensure OTEL is NOT enabled via env var
			origOtelEnabled := os.Getenv("ADDT_OTEL_ENABLED")
			os.Unsetenv("ADDT_OTEL_ENABLED")
			defer func() {
				if origOtelEnabled != "" {
					os.Setenv("ADDT_OTEL_ENABLED", origOtelEnabled)
				}
			}()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo ENDPOINT:${OTEL_EXPORTER_OTLP_ENDPOINT:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			endpoint := extractMarker(output, "ENDPOINT:")
			if endpoint != "NOTSET" {
				t.Errorf("Expected OTEL_EXPORTER_OTLP_ENDPOINT to be NOTSET when disabled, got %q", endpoint)
			}
		})
	}
}

func TestOtel_Addt_ResourceAttributes(t *testing.T) {
	// Scenario: User enables OTEL. The OTEL_RESOURCE_ATTRIBUTES env var
	// should contain runtime context: addt.extension and addt.provider.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
otel:
  enabled: true
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			defer saveRestoreEnv(t, "ADDT_OTEL_ENABLED", "true")()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo RESATTR:${OTEL_RESOURCE_ATTRIBUTES:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			resAttrs := extractMarker(output, "RESATTR:")
			if resAttrs == "NOTSET" || resAttrs == "" {
				t.Fatalf("Expected OTEL_RESOURCE_ATTRIBUTES to be set, got %q", resAttrs)
			}

			if !strings.Contains(resAttrs, "addt.extension=") {
				t.Errorf("Expected OTEL_RESOURCE_ATTRIBUTES to contain addt.extension=, got %q", resAttrs)
			}

			if !strings.Contains(resAttrs, "addt.provider="+prov) {
				t.Errorf("Expected OTEL_RESOURCE_ATTRIBUTES to contain addt.provider=%s, got %q", prov, resAttrs)
			}
		})
	}
}

func TestOtel_Addt_HeadersForwarded(t *testing.T) {
	// Scenario: User sets otel.headers to forward custom OTLP headers.
	// The OTEL_EXPORTER_OTLP_HEADERS env var should be set in the container.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			addtHome := t.TempDir()
			defer saveRestoreEnv(t, "ADDT_HOME", addtHome)()

			dir, cleanup := setupAddtDirWithExtensions(t, prov, `
otel:
  enabled: true
  headers: "Authorization=Bearer token123"
`)
			defer cleanup()
			ensureAddtImage(t, dir, "debug")

			defer saveRestoreEnv(t, "ADDT_OTEL_ENABLED", "true")()
			defer saveRestoreEnv(t, "ADDT_OTEL_HEADERS", "Authorization=Bearer token123")()

			output, err := runRunSubcommand(t, dir, "debug",
				"-c", "echo HEADERS:${OTEL_EXPORTER_OTLP_HEADERS:-NOTSET}")
			t.Logf("Output:\n%s", output)
			if err != nil {
				t.Fatalf("run failed: %v\nOutput:\n%s", err, output)
			}

			headers := extractMarker(output, "HEADERS:")
			if headers != "Authorization=Bearer token123" {
				t.Errorf("Expected OTEL_EXPORTER_OTLP_HEADERS=Authorization=Bearer token123, got %q", headers)
			}
		})
	}
}
