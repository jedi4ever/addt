package otel

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Enabled != false {
		t.Errorf("Expected Enabled=false, got %v", cfg.Enabled)
	}
	if cfg.Endpoint != "http://host.docker.internal:4318" {
		t.Errorf("Expected Endpoint=http://host.docker.internal:4318, got %v", cfg.Endpoint)
	}
	if cfg.Protocol != "http/json" {
		t.Errorf("Expected Protocol=http/json, got %v", cfg.Protocol)
	}
	if cfg.ServiceName != "addt" {
		t.Errorf("Expected ServiceName=addt, got %v", cfg.ServiceName)
	}
}

func TestApplySettings(t *testing.T) {
	cfg := DefaultConfig()

	enabled := true
	endpoint := "http://otel.example.com:4317"
	protocol := "grpc"
	serviceName := "my-service"
	headers := "key=value"

	settings := &Settings{
		Enabled:     &enabled,
		Endpoint:    &endpoint,
		Protocol:    &protocol,
		ServiceName: &serviceName,
		Headers:     &headers,
	}

	applySettings(&cfg, settings)

	if cfg.Enabled != true {
		t.Errorf("Expected Enabled=true, got %v", cfg.Enabled)
	}
	if cfg.Endpoint != endpoint {
		t.Errorf("Expected Endpoint=%s, got %s", endpoint, cfg.Endpoint)
	}
	if cfg.Protocol != protocol {
		t.Errorf("Expected Protocol=%s, got %s", protocol, cfg.Protocol)
	}
	if cfg.ServiceName != serviceName {
		t.Errorf("Expected ServiceName=%s, got %s", serviceName, cfg.ServiceName)
	}
	if cfg.Headers != headers {
		t.Errorf("Expected Headers=%s, got %s", headers, cfg.Headers)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	// Save and restore environment
	envVars := []string{
		"ADDT_OTEL_ENABLED",
		"ADDT_OTEL_ENDPOINT",
		"ADDT_OTEL_PROTOCOL",
		"ADDT_OTEL_SERVICE_NAME",
		"ADDT_OTEL_HEADERS",
	}
	saved := make(map[string]string)
	for _, key := range envVars {
		saved[key] = os.Getenv(key)
	}
	defer func() {
		for key, val := range saved {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Set test values
	os.Setenv("ADDT_OTEL_ENABLED", "true")
	os.Setenv("ADDT_OTEL_ENDPOINT", "http://env-endpoint:4317")
	os.Setenv("ADDT_OTEL_PROTOCOL", "grpc")
	os.Setenv("ADDT_OTEL_SERVICE_NAME", "env-service")
	os.Setenv("ADDT_OTEL_HEADERS", "auth=token123")

	cfg := DefaultConfig()
	applyEnvOverrides(&cfg)

	if cfg.Enabled != true {
		t.Errorf("Expected Enabled=true from env, got %v", cfg.Enabled)
	}
	if cfg.Endpoint != "http://env-endpoint:4317" {
		t.Errorf("Expected Endpoint from env, got %s", cfg.Endpoint)
	}
	if cfg.Protocol != "grpc" {
		t.Errorf("Expected Protocol=grpc from env, got %s", cfg.Protocol)
	}
	if cfg.ServiceName != "env-service" {
		t.Errorf("Expected ServiceName from env, got %s", cfg.ServiceName)
	}
	if cfg.Headers != "auth=token123" {
		t.Errorf("Expected Headers from env, got %s", cfg.Headers)
	}
}

func TestGetEnvVars(t *testing.T) {
	emptyAttrs := ResourceAttrs{}

	// Test when disabled
	cfg := Config{Enabled: false}
	env := GetEnvVars(cfg, emptyAttrs)
	if env != nil {
		t.Errorf("Expected nil env vars when disabled, got %v", env)
	}

	// Test when enabled with custom service name (no override)
	cfg = Config{
		Enabled:     true,
		Endpoint:    "http://otel:4318",
		Protocol:    "http/protobuf",
		ServiceName: "test-service",
		Headers:     "key=value",
	}
	env = GetEnvVars(cfg, emptyAttrs)

	if env["OTEL_EXPORTER_OTLP_ENDPOINT"] != cfg.Endpoint {
		t.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT=%s, want %s", env["OTEL_EXPORTER_OTLP_ENDPOINT"], cfg.Endpoint)
	}
	if env["OTEL_EXPORTER_OTLP_PROTOCOL"] != cfg.Protocol {
		t.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL=%s, want %s", env["OTEL_EXPORTER_OTLP_PROTOCOL"], cfg.Protocol)
	}
	if env["OTEL_SERVICE_NAME"] != "test-service" {
		t.Errorf("OTEL_SERVICE_NAME=%s, want test-service", env["OTEL_SERVICE_NAME"])
	}
	if env["OTEL_EXPORTER_OTLP_HEADERS"] != cfg.Headers {
		t.Errorf("OTEL_EXPORTER_OTLP_HEADERS=%s, want %s", env["OTEL_EXPORTER_OTLP_HEADERS"], cfg.Headers)
	}
	if env["CLAUDE_CODE_ENABLE_TELEMETRY"] != "1" {
		t.Errorf("CLAUDE_CODE_ENABLE_TELEMETRY=%s, want 1", env["CLAUDE_CODE_ENABLE_TELEMETRY"])
	}
	if env["OTEL_METRICS_EXPORTER"] != "otlp" {
		t.Errorf("OTEL_METRICS_EXPORTER=%s, want otlp", env["OTEL_METRICS_EXPORTER"])
	}
	if env["OTEL_LOGS_EXPORTER"] != "otlp" {
		t.Errorf("OTEL_LOGS_EXPORTER=%s, want otlp", env["OTEL_LOGS_EXPORTER"])
	}

	// Test without headers
	cfg.Headers = ""
	env = GetEnvVars(cfg, emptyAttrs)
	if _, ok := env["OTEL_EXPORTER_OTLP_HEADERS"]; ok {
		t.Error("Expected no OTEL_EXPORTER_OTLP_HEADERS when empty")
	}
}

func TestGetEnvVars_ServiceNameWithExtension(t *testing.T) {
	// Default service name "addt" gets extension appended
	cfg := Config{Enabled: true, Endpoint: "http://otel:4318", Protocol: "http/json", ServiceName: "addt"}
	attrs := ResourceAttrs{Extension: "claude"}
	env := GetEnvVars(cfg, attrs)

	if env["OTEL_SERVICE_NAME"] != "addt-claude" {
		t.Errorf("OTEL_SERVICE_NAME=%s, want addt-claude", env["OTEL_SERVICE_NAME"])
	}

	// Custom service name is NOT overridden
	cfg.ServiceName = "my-service"
	env = GetEnvVars(cfg, attrs)
	if env["OTEL_SERVICE_NAME"] != "my-service" {
		t.Errorf("OTEL_SERVICE_NAME=%s, want my-service", env["OTEL_SERVICE_NAME"])
	}
}

func TestGetEnvVars_ResourceAttributes(t *testing.T) {
	cfg := Config{Enabled: true, Endpoint: "http://otel:4318", Protocol: "http/json", ServiceName: "addt"}
	attrs := ResourceAttrs{
		Extension: "claude",
		Provider:  "podman",
		Version:   "0.0.9",
		Project:   "myproject",
	}
	env := GetEnvVars(cfg, attrs)

	ra := env["OTEL_RESOURCE_ATTRIBUTES"]
	if ra == "" {
		t.Fatal("OTEL_RESOURCE_ATTRIBUTES not set")
	}
	for _, want := range []string{"addt.extension=claude", "addt.provider=podman", "addt.version=0.0.9", "addt.project=myproject"} {
		if !strings.Contains(ra, want) {
			t.Errorf("OTEL_RESOURCE_ATTRIBUTES=%q, missing %q", ra, want)
		}
	}
}

func TestGetEnvVars_EmptyResourceAttributes(t *testing.T) {
	cfg := Config{Enabled: true, Endpoint: "http://otel:4318", Protocol: "http/json", ServiceName: "addt"}
	env := GetEnvVars(cfg, ResourceAttrs{})

	if _, ok := env["OTEL_RESOURCE_ATTRIBUTES"]; ok {
		t.Error("OTEL_RESOURCE_ATTRIBUTES should not be set when attrs are empty")
	}
}

func TestLoadConfig(t *testing.T) {
	// Test with nil settings
	cfg := LoadConfig(nil, nil)
	defaults := DefaultConfig()
	if cfg.Enabled != defaults.Enabled {
		t.Errorf("Expected default Enabled, got %v", cfg.Enabled)
	}

	// Test project settings override global
	globalEnabled := false
	projectEnabled := true
	globalEndpoint := "http://global:4318"
	projectEndpoint := "http://project:4318"

	global := &Settings{Enabled: &globalEnabled, Endpoint: &globalEndpoint}
	project := &Settings{Enabled: &projectEnabled, Endpoint: &projectEndpoint}

	cfg = LoadConfig(global, project)
	if cfg.Enabled != true {
		t.Errorf("Expected project setting to override global, got Enabled=%v", cfg.Enabled)
	}
	if cfg.Endpoint != projectEndpoint {
		t.Errorf("Expected project endpoint, got %s", cfg.Endpoint)
	}
}
