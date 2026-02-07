//go:build addt

package addt

import (
	"strings"
	"testing"

	configcmd "github.com/jedi4ever/addt/cmd/config"
)

// --- Config tests (in-process, no container needed) ---

func TestPorts_Addt_ConfigLoaded(t *testing.T) {
	// Scenario: User sets ports.expose and ports.range_start in project config,
	// then verifies both appear in config list with correct values and source.
	_, cleanup := setupAddtDir(t, "", `
ports:
  expose:
    - "3000"
    - "8080"
  range_start: 31000
`)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	if !strings.Contains(output, "ports.expose") {
		t.Errorf("Expected output to contain ports.expose, got:\n%s", output)
	}
	if !strings.Contains(output, "ports.range_start") {
		t.Errorf("Expected output to contain ports.range_start, got:\n%s", output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "ports.expose") {
			if !strings.Contains(line, "3000") || !strings.Contains(line, "8080") {
				t.Errorf("Expected ports.expose to contain 3000 and 8080, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected ports.expose source=project, got line: %s", line)
			}
		}
		if strings.Contains(line, "ports.range_start") {
			if !strings.Contains(line, "31000") {
				t.Errorf("Expected ports.range_start=31000, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected ports.range_start source=project, got line: %s", line)
			}
		}
	}
}

func TestPorts_Addt_ConfigViaSet(t *testing.T) {
	// Scenario: User enables ports via 'config set ports.expose 3000,8080',
	// then verifies they appear in config list.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	// Set ports.expose via config set
	captureOutput(t, func() {
		configcmd.HandleCommand([]string{"set", "ports.expose", "3000,8080"})
	})

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "ports.expose") {
			if !strings.Contains(line, "3000") || !strings.Contains(line, "8080") {
				t.Errorf("Expected ports.expose to contain 3000 and 8080 after config set, got line: %s", line)
			}
			if !strings.Contains(line, "project") {
				t.Errorf("Expected ports.expose source=project after config set, got line: %s", line)
			}
		}
	}
}

func TestPorts_Addt_DefaultValues(t *testing.T) {
	// Scenario: User starts with no ports config and checks defaults.
	// ports.forward should default to true.
	_, cleanup := setupAddtDir(t, "", ``)
	defer cleanup()

	output := captureOutput(t, func() {
		configcmd.HandleCommand([]string{"list"})
	})

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "ports.forward") {
			if !strings.Contains(line, "true") {
				t.Errorf("Expected ports.forward default=true, got line: %s", line)
			}
			if !strings.Contains(line, "default") {
				t.Errorf("Expected ports.forward source=default, got line: %s", line)
			}
		}
	}
}

// --- Container tests (subprocess, both providers) ---

// extractPortMap finds the PORT_MAP value in subprocess output.
// We use PORT_MAP: markers to isolate it from test framework noise.
func extractPortMap(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PORT_MAP:") {
			return strings.TrimPrefix(line, "PORT_MAP:")
		}
	}
	return ""
}

// extractPrompt finds the PROMPT value in subprocess output.
func extractPrompt(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PROMPT:") {
			return strings.TrimPrefix(line, "PROMPT:")
		}
	}
	return ""
}

// extractPortResult finds the PORT_RESULT value in subprocess output.
func extractPortResult(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PORT_RESULT:") {
			return strings.TrimPrefix(line, "PORT_RESULT:")
		}
	}
	return ""
}

func TestPorts_Addt_PortMapEnvVar(t *testing.T) {
	// Scenario: User configures ports.expose with two ports and a custom range_start.
	// Inside the container, ADDT_PORT_MAP should be set with mappings for both ports.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
ports:
  expose:
    - "3000"
    - "8080"
  range_start: 30000
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, _ := runShellCommand(t, dir,
				"claude", "-c",
				"echo PORT_MAP:$ADDT_PORT_MAP")

			portMap := extractPortMap(output)
			t.Logf("Port map: %q", portMap)

			if portMap == "" {
				t.Errorf("Expected ADDT_PORT_MAP to be set, but it was empty.\nFull output:\n%s", output)
			}
			if !strings.Contains(portMap, "3000:") {
				t.Errorf("Expected port map to contain 3000: mapping, got: %s", portMap)
			}
			if !strings.Contains(portMap, "8080:") {
				t.Errorf("Expected port map to contain 8080: mapping, got: %s", portMap)
			}
		})
	}
}

func TestPorts_Addt_SystemPromptInjected(t *testing.T) {
	// Scenario: User configures ports with system prompt injection enabled (default).
	// The entrypoint should generate ADDT_SYSTEM_PROMPT containing port mapping info
	// that guides the AI to communicate correct host ports to users.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
ports:
  expose:
    - "3000"
    - "8080"
  range_start: 30000
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, _ := runShellCommand(t, dir,
				"claude", "-c",
				"echo PROMPT:$ADDT_SYSTEM_PROMPT")

			prompt := extractPrompt(output)
			t.Logf("System prompt: %q", prompt)

			if prompt == "" {
				t.Errorf("Expected ADDT_SYSTEM_PROMPT to be set, but it was empty.\nFull output:\n%s", output)
			}
			if !strings.Contains(prompt, "Port Mapping") {
				t.Errorf("Expected system prompt to contain 'Port Mapping', got: %s", prompt)
			}
			if !strings.Contains(prompt, "3000") {
				t.Errorf("Expected system prompt to reference port 3000, got: %s", prompt)
			}
			if !strings.Contains(prompt, "8080") {
				t.Errorf("Expected system prompt to reference port 8080, got: %s", prompt)
			}
		})
	}
}

func TestPorts_Addt_ServiceAccessible(t *testing.T) {
	// Scenario: User exposes port 8000. Inside the container, a Python HTTP server
	// is started on that port, then curl verifies it's accessible from localhost.
	// This confirms the port forwarding pipeline works end-to-end.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
ports:
  expose:
    - "8000"
  range_start: 30000
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, _ := runShellCommand(t, dir,
				"claude", "-c",
				"python3 -m http.server 8000 & sleep 1; CODE=$(curl -s --connect-timeout 3 -o /dev/null -w '%{http_code}' http://localhost:8000); kill %1 2>/dev/null; echo PORT_RESULT:$CODE")

			result := extractPortResult(output)
			t.Logf("Service accessible result: %q", result)

			if result == "" {
				t.Errorf("Shell command did not produce PORT_RESULT — entrypoint may have failed.\nFull output:\n%s", output)
			} else if result != "200" {
				t.Errorf("Expected PORT_RESULT:200 (service reachable on exposed port), got: %s", result)
			}
		})
	}
}

func TestPorts_Addt_NoPorts(t *testing.T) {
	// Scenario: User configures no ports (ports.expose is empty).
	// ADDT_PORT_MAP should not be set inside the container.
	providers := requireProviders(t)

	for _, prov := range providers {
		t.Run(prov, func(t *testing.T) {
			dir, cleanup := setupAddtDir(t, prov, `
ports:
  expose: []
`)
			defer cleanup()
			ensureAddtImage(t, dir, "claude")

			output, _ := runShellCommand(t, dir,
				"claude", "-c",
				"echo PORT_MAP:${ADDT_PORT_MAP:-NONE}")

			portMap := extractPortMap(output)
			t.Logf("No-ports result: %q", portMap)

			if portMap == "" {
				t.Errorf("Shell command did not produce PORT_MAP — entrypoint may have failed.\nFull output:\n%s", output)
			} else if portMap != "NONE" {
				t.Errorf("Expected PORT_MAP:NONE when no ports configured, got: %s", portMap)
			}
		})
	}
}
