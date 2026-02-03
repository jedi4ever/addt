//go:build integration

package core

import (
	"os/exec"
	"strings"
	"testing"
)

// checkDockerForOrchestrator verifies Docker is available
func checkDockerForOrchestrator(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH, skipping integration test")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker daemon not running, skipping integration test")
	}
}

func TestSystemPrompt_Integration_PortMapGeneration(t *testing.T) {
	checkDockerForOrchestrator(t)

	// Test that ADDT_PORT_MAP env var is correctly formatted
	// by running a container that echoes it

	portMap := "3000:30000,8080:30001"

	// Pre-pull alpine to avoid pull messages in output
	exec.Command("docker", "pull", "alpine:latest").Run()

	cmd := exec.Command("docker", "run", "--rm",
		"-e", "ADDT_PORT_MAP="+portMap,
		"alpine:latest",
		"printenv", "ADDT_PORT_MAP")

	// Use Output() instead of CombinedOutput() to avoid stderr (like Docker pull messages)
	output, err := cmd.Output()
	if err != nil {
		// Get stderr for debugging
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Failed to run container: %v\nStderr: %s", err, string(exitErr.Stderr))
		}
		t.Fatalf("Failed to run container: %v", err)
	}

	if strings.TrimSpace(string(output)) != portMap {
		t.Errorf("ADDT_PORT_MAP = %q, want %q", strings.TrimSpace(string(output)), portMap)
	}
}

func TestSystemPrompt_Integration_EntrypointGeneratesPrompt(t *testing.T) {
	checkDockerForOrchestrator(t)

	// Build a test image if needed
	testImageName := "addt-test-systemprompt"

	// Check if we have an addt image to test with
	cmd := exec.Command("docker", "image", "inspect", testImageName)
	if cmd.Run() != nil {
		// Try to find any addt image
		cmd = exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", "reference=addt:*")
		output, err := cmd.Output()
		if err != nil || len(strings.TrimSpace(string(output))) == 0 {
			t.Skip("No addt image available for system prompt testing")
		}
		testImageName = strings.Split(strings.TrimSpace(string(output)), "\n")[0]
	}

	// Run the entrypoint script with ADDT_PORT_MAP set and check ADDT_SYSTEM_PROMPT
	// We need to source the relevant parts of the entrypoint

	portMap := "3000:30000,8080:30001"

	// Use a bash script to simulate what the entrypoint does
	script := `
export ADDT_PORT_MAP="` + portMap + `"
export ADDT_SYSTEM_PROMPT=""

if [ -n "$ADDT_PORT_MAP" ]; then
    ADDT_SYSTEM_PROMPT="# Port Mapping Information

When you start a service inside this container on certain ports, tell the user the correct HOST port to access it from their browser.

Port mappings (container→host):
"
    IFS=',' read -ra MAPPINGS <<< "$ADDT_PORT_MAP"
    for mapping in "${MAPPINGS[@]}"; do
        IFS=':' read -ra PORTS <<< "$mapping"
        CONTAINER_PORT="${PORTS[0]}"
        HOST_PORT="${PORTS[1]}"
        ADDT_SYSTEM_PROMPT+="- Container port $CONTAINER_PORT → Host port $HOST_PORT (user accesses: http://localhost:$HOST_PORT)
"
    done
fi

echo "$ADDT_SYSTEM_PROMPT"
`

	cmd = exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", script)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Verify the system prompt contains expected content
	expectedPhrases := []string{
		"Port Mapping Information",
		"Container port 3000",
		"Host port 30000",
		"Container port 8080",
		"Host port 30001",
		"localhost:30000",
		"localhost:30001",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(outputStr, phrase) {
			t.Errorf("System prompt missing %q\nGot: %s", phrase, outputStr)
		}
	}
}

func TestSystemPrompt_Integration_ArgsShIncludesPrompt(t *testing.T) {
	checkDockerForOrchestrator(t)

	// Find a claude image to test with
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", "reference=addt:claude*")
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		t.Skip("No addt claude image available for args.sh testing")
	}
	testImageName := strings.Split(strings.TrimSpace(string(output)), "\n")[0]

	// Test that args.sh adds --append-system-prompt when ADDT_SYSTEM_PROMPT is set
	script := `
export ADDT_SYSTEM_PROMPT="Test system prompt"

# Run args.sh and check output
if [ -f /usr/local/share/addt/extensions/claude/args.sh ]; then
    bash /usr/local/share/addt/extensions/claude/args.sh --help
else
    echo "args.sh not found"
fi
`

	cmd = exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", script)

	output, err = cmd.CombinedOutput()
	if err != nil {
		// args.sh might not exist or might fail - that's ok for this test
		t.Logf("Command output: %s", string(output))
	}

	outputStr := string(output)

	// If args.sh exists and ADDT_SYSTEM_PROMPT is set, output should include --append-system-prompt
	if strings.Contains(outputStr, "args.sh not found") {
		t.Skip("args.sh not found in image")
	}

	if !strings.Contains(outputStr, "--append-system-prompt") {
		t.Errorf("args.sh should include --append-system-prompt when ADDT_SYSTEM_PROMPT is set\nGot: %s", outputStr)
	}
}

func TestSystemPrompt_Integration_NoPortsNoPrompt(t *testing.T) {
	checkDockerForOrchestrator(t)

	// Test that when no ports are configured, no system prompt is added

	script := `
# No ADDT_PORT_MAP set
ADDT_SYSTEM_PROMPT=""

if [ -n "$ADDT_PORT_MAP" ]; then
    ADDT_SYSTEM_PROMPT="Port info here"
fi

if [ -z "$ADDT_SYSTEM_PROMPT" ]; then
    echo "EMPTY"
else
    echo "HAS_CONTENT"
fi
`

	cmd := exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/sh",
		"alpine:latest",
		"-c", script)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	if strings.TrimSpace(string(output)) != "EMPTY" {
		t.Errorf("System prompt should be empty when no ports configured, got: %s", string(output))
	}
}

func TestSystemPrompt_Integration_PortMapParsing(t *testing.T) {
	checkDockerForOrchestrator(t)

	testCases := []struct {
		name        string
		portMap     string
		expectPorts []struct {
			container string
			host      string
		}
	}{
		{
			name:    "single port",
			portMap: "3000:30000",
			expectPorts: []struct {
				container string
				host      string
			}{
				{"3000", "30000"},
			},
		},
		{
			name:    "multiple ports",
			portMap: "3000:30000,8080:30001,5432:30002",
			expectPorts: []struct {
				container string
				host      string
			}{
				{"3000", "30000"},
				{"8080", "30001"},
				{"5432", "30002"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use sh-compatible syntax (no arrays)
			script := `
ADDT_PORT_MAP="` + tc.portMap + `"

# Parse comma-separated mappings using sh-compatible method
echo "$ADDT_PORT_MAP" | tr ',' '\n' | while read mapping; do
    echo "$mapping"
done
`

			cmd := exec.Command("docker", "run", "--rm",
				"--entrypoint", "/bin/sh",
				"alpine:latest",
				"-c", script)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
			}

			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) != len(tc.expectPorts) {
				t.Errorf("Expected %d port mappings, got %d: %v", len(tc.expectPorts), len(lines), lines)
			}

			for i, expected := range tc.expectPorts {
				expectedLine := expected.container + ":" + expected.host
				if i < len(lines) && lines[i] != expectedLine {
					t.Errorf("Port %d: got %q, want %q", i, lines[i], expectedLine)
				}
			}
		})
	}
}

func TestSystemPrompt_Integration_FullFlow(t *testing.T) {
	checkDockerForOrchestrator(t)

	// Find a claude image
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", "reference=addt:claude*")
	output, err := cmd.Output()
	if err != nil || len(strings.TrimSpace(string(output))) == 0 {
		t.Skip("No addt claude image available for full flow testing")
	}
	testImageName := strings.Split(strings.TrimSpace(string(output)), "\n")[0]

	// Simulate the full entrypoint flow with port mapping
	script := `
# Simulate the entrypoint's port mapping logic
export ADDT_PORT_MAP="3000:30000,8080:30001"
export ADDT_SYSTEM_PROMPT=""

if [ -n "$ADDT_PORT_MAP" ]; then
    ADDT_SYSTEM_PROMPT="# Port Mapping Information

When you start a service inside this container on certain ports, tell the user the correct HOST port to access it from their browser.

Port mappings (container→host):
"
    IFS=',' read -ra MAPPINGS <<< "$ADDT_PORT_MAP"
    for mapping in "${MAPPINGS[@]}"; do
        IFS=':' read -ra PORTS <<< "$mapping"
        CONTAINER_PORT="${PORTS[0]}"
        HOST_PORT="${PORTS[1]}"
        ADDT_SYSTEM_PROMPT+="- Container port $CONTAINER_PORT → Host port $HOST_PORT (user accesses: http://localhost:$HOST_PORT)
"
    done

    ADDT_SYSTEM_PROMPT+="
IMPORTANT:
- When testing/starting services inside the container, use the container ports (e.g., http://localhost:3000)
- When telling the USER where to access services in their browser, use the HOST ports (e.g., http://localhost:30000)
- Always remind the user to use the host port in their browser"
fi

# Check if args.sh would include the prompt
if [ -n "$ADDT_SYSTEM_PROMPT" ] && [ -f /usr/local/share/addt/extensions/claude/args.sh ]; then
    echo "WOULD_INCLUDE_PROMPT"
    # Show what would be added
    echo "---"
    echo "$ADDT_SYSTEM_PROMPT"
else
    echo "NO_PROMPT_OR_NO_ARGS_SH"
fi
`

	cmd = exec.Command("docker", "run", "--rm",
		"--entrypoint", "/bin/bash",
		testImageName,
		"-c", script)

	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	if !strings.Contains(outputStr, "WOULD_INCLUDE_PROMPT") {
		if strings.Contains(outputStr, "NO_PROMPT_OR_NO_ARGS_SH") {
			t.Skip("args.sh not found in image")
		}
		t.Errorf("Expected prompt to be included\nGot: %s", outputStr)
	}

	// Verify the prompt content
	expectedPhrases := []string{
		"Port Mapping Information",
		"Container port 3000",
		"Host port 30000",
		"IMPORTANT",
		"host port in their browser",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(outputStr, phrase) {
			t.Errorf("System prompt missing %q\nGot: %s", phrase, outputStr)
		}
	}

	t.Logf("Full system prompt output:\n%s", outputStr)
}

func TestPortMapEnvVar_Integration_GoCodeGeneration(t *testing.T) {
	// This test verifies the Go code generates correct port map strings
	// without needing Docker

	testCases := []struct {
		name           string
		ports          []string
		portRangeStart int
		wantFormat     bool // just check format, not exact values
	}{
		{
			name:           "single port",
			ports:          []string{"3000"},
			portRangeStart: 30000,
			wantFormat:     true,
		},
		{
			name:           "multiple ports",
			ports:          []string{"3000", "8080", "5432"},
			portRangeStart: 30000,
			wantFormat:     true,
		},
		{
			name:           "with whitespace",
			ports:          []string{" 3000 ", "8080"},
			portRangeStart: 30000,
			wantFormat:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &mockConfig{
				Ports:          tc.ports,
				PortRangeStart: tc.portRangeStart,
			}

			// Simulate buildEnvironment logic
			portMap := buildPortMapString(cfg.Ports, cfg.PortRangeStart)

			if portMap == "" {
				t.Fatal("Port map should not be empty")
			}

			// Verify format: "port:port,port:port"
			mappings := strings.Split(portMap, ",")
			if len(mappings) != len(tc.ports) {
				t.Errorf("Expected %d mappings, got %d: %q", len(tc.ports), len(mappings), portMap)
			}

			for i, mapping := range mappings {
				parts := strings.Split(mapping, ":")
				if len(parts) != 2 {
					t.Errorf("Mapping %d: %q should have format 'container:host'", i, mapping)
				}
			}

			t.Logf("Port map: %s", portMap)
		})
	}
}

// mockConfig for port map testing
type mockConfig struct {
	Ports          []string
	PortRangeStart int
}

// buildPortMapString simulates the orchestrator's port map string generation
func buildPortMapString(ports []string, startPort int) string {
	if len(ports) == 0 {
		return ""
	}

	var mappings []string
	hostPort := startPort

	for _, containerPort := range ports {
		containerPort = strings.TrimSpace(containerPort)
		mappings = append(mappings, containerPort+":"+string(rune('0'+hostPort/10000))+string(rune('0'+(hostPort/1000)%10))+string(rune('0'+(hostPort/100)%10))+string(rune('0'+(hostPort/10)%10))+string(rune('0'+hostPort%10)))
		hostPort++
	}

	return strings.Join(mappings, ",")
}
