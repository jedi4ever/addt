package core

import (
	"strings"
	"testing"

	"github.com/jedi4ever/addt/provider"
)

// mockEnvProvider implements the minimal provider interface for env tests
type mockEnvProvider struct{}

func (m *mockEnvProvider) Initialize(cfg *provider.Config) error              { return nil }
func (m *mockEnvProvider) Run(spec *provider.RunSpec) error                   { return nil }
func (m *mockEnvProvider) Shell(spec *provider.RunSpec) error                 { return nil }
func (m *mockEnvProvider) Cleanup() error                                     { return nil }
func (m *mockEnvProvider) Exists(name string) bool                            { return false }
func (m *mockEnvProvider) IsRunning(name string) bool                         { return false }
func (m *mockEnvProvider) Start(name string) error                            { return nil }
func (m *mockEnvProvider) Stop(name string) error                             { return nil }
func (m *mockEnvProvider) Remove(name string) error                           { return nil }
func (m *mockEnvProvider) List() ([]provider.Environment, error)              { return nil, nil }
func (m *mockEnvProvider) GeneratePersistentName() string                     { return "test-persistent" }
func (m *mockEnvProvider) GenerateEphemeralName() string                      { return "test-ephemeral" }
func (m *mockEnvProvider) GetStatus(cfg *provider.Config, name string) string { return "test" }
func (m *mockEnvProvider) GetName() string                                    { return "mock" }
func (m *mockEnvProvider) GetExtensionEnvVars(imageName string) []string      { return nil }
func (m *mockEnvProvider) DetermineImageName() string                         { return "test-image" }
func (m *mockEnvProvider) BuildIfNeeded(rebuild bool, rebuildBase bool) error { return nil }

func TestBuildEnvironment_Basic(t *testing.T) {
	cfg := &provider.Config{}

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	// COLUMNS and LINES should always be set
	if env["COLUMNS"] == "" {
		t.Error("COLUMNS not set")
	}

	if env["LINES"] == "" {
		t.Error("LINES not set")
	}
}

func TestBuildEnvironment_Firewall(t *testing.T) {
	cfg := &provider.Config{
		FirewallEnabled: true,
		FirewallMode:    "allowlist",
	}

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if env["ADDT_FIREWALL_ENABLED"] != "true" {
		t.Errorf("ADDT_FIREWALL_ENABLED = %q, want 'true'", env["ADDT_FIREWALL_ENABLED"])
	}

	if env["ADDT_FIREWALL_MODE"] != "allowlist" {
		t.Errorf("ADDT_FIREWALL_MODE = %q, want 'allowlist'", env["ADDT_FIREWALL_MODE"])
	}
}

func TestBuildEnvironment_FirewallDisabled(t *testing.T) {
	cfg := &provider.Config{
		FirewallEnabled: false,
	}

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if _, ok := env["ADDT_FIREWALL_ENABLED"]; ok {
		t.Error("ADDT_FIREWALL_ENABLED should not be set when firewall is disabled")
	}
}

func TestBuildEnvironment_Command(t *testing.T) {
	cfg := &provider.Config{
		Command: "codex",
	}

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if env["ADDT_COMMAND"] != "codex" {
		t.Errorf("ADDT_COMMAND = %q, want 'codex'", env["ADDT_COMMAND"])
	}
}

func TestBuildEnvironment_CommandNotSet(t *testing.T) {
	cfg := &provider.Config{
		Command: "",
	}

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if _, ok := env["ADDT_COMMAND"]; ok {
		t.Error("ADDT_COMMAND should not be set when command is empty")
	}
}

func TestBuildEnvironment_PortMap(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{"3000", "8080"},
		PortRangeStart: 30000,
	}

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if env["ADDT_PORT_MAP"] == "" {
		t.Error("ADDT_PORT_MAP not set")
	}
}

func TestBuildEnvironment_NoPortMap(t *testing.T) {
	cfg := &provider.Config{
		Ports:          []string{},
		PortRangeStart: 30000,
	}

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if _, ok := env["ADDT_PORT_MAP"]; ok {
		t.Error("ADDT_PORT_MAP should not be set when no ports configured")
	}
}

func TestBuildEnvironment_OtelEnabled(t *testing.T) {
	cfg := &provider.Config{}
	cfg.Otel.Enabled = true
	cfg.Otel.Endpoint = "http://host.docker.internal:4318"
	cfg.Otel.Protocol = "http/json"
	cfg.Otel.ServiceName = "addt"
	cfg.Extensions = "claude"
	cfg.Provider = "podman"
	cfg.AddtVersion = "0.0.9"

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if env["OTEL_EXPORTER_OTLP_ENDPOINT"] != "http://host.docker.internal:4318" {
		t.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT = %q, want 'http://host.docker.internal:4318'", env["OTEL_EXPORTER_OTLP_ENDPOINT"])
	}
	if env["OTEL_EXPORTER_OTLP_PROTOCOL"] != "http/json" {
		t.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL = %q, want 'http/json'", env["OTEL_EXPORTER_OTLP_PROTOCOL"])
	}
	if env["OTEL_SERVICE_NAME"] != "addt-claude" {
		t.Errorf("OTEL_SERVICE_NAME = %q, want 'addt-claude'", env["OTEL_SERVICE_NAME"])
	}
	if env["CLAUDE_CODE_ENABLE_TELEMETRY"] != "1" {
		t.Errorf("CLAUDE_CODE_ENABLE_TELEMETRY = %q, want '1'", env["CLAUDE_CODE_ENABLE_TELEMETRY"])
	}
	ra := env["OTEL_RESOURCE_ATTRIBUTES"]
	if ra == "" {
		t.Fatal("OTEL_RESOURCE_ATTRIBUTES not set")
	}
	if !strings.Contains(ra, "addt.extension=claude") {
		t.Errorf("OTEL_RESOURCE_ATTRIBUTES = %q, missing addt.extension=claude", ra)
	}
	if !strings.Contains(ra, "addt.provider=podman") {
		t.Errorf("OTEL_RESOURCE_ATTRIBUTES = %q, missing addt.provider=podman", ra)
	}
}

func TestBuildEnvironment_OtelDisabled(t *testing.T) {
	cfg := &provider.Config{}
	cfg.Otel.Enabled = false

	env := BuildEnvironment(&mockEnvProvider{}, cfg)

	if _, ok := env["OTEL_EXPORTER_OTLP_ENDPOINT"]; ok {
		t.Error("OTEL_EXPORTER_OTLP_ENDPOINT should not be set when OTEL is disabled")
	}
}

func TestParseEnvVarSpec(t *testing.T) {
	tests := []struct {
		spec        string
		wantName    string
		wantDefault string
	}{
		{"VAR_NAME", "VAR_NAME", ""},
		{"VAR_NAME=value", "VAR_NAME", "value"},
		{"VAR_NAME=", "VAR_NAME", ""},
		{"VAR_NAME=value=with=equals", "VAR_NAME", "value=with=equals"},
		{"CLAUDE_CODE_ENABLE_TELEMETRY=1", "CLAUDE_CODE_ENABLE_TELEMETRY", "1"},
		{"OTEL_LOG_USER_PROMPTS=true", "OTEL_LOG_USER_PROMPTS", "true"},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			name, defaultValue := parseEnvVarSpec(tt.spec)
			if name != tt.wantName {
				t.Errorf("parseEnvVarSpec(%q) name = %q, want %q", tt.spec, name, tt.wantName)
			}
			if defaultValue != tt.wantDefault {
				t.Errorf("parseEnvVarSpec(%q) default = %q, want %q", tt.spec, defaultValue, tt.wantDefault)
			}
		})
	}
}
