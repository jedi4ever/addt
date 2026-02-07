package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/config/otel"
)

func TestOtelKeyValidation(t *testing.T) {
	otelKeys := []string{
		"otel.enabled", "otel.endpoint", "otel.protocol",
		"otel.service_name", "otel.headers",
	}

	for _, key := range otelKeys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestOtelGetValue(t *testing.T) {
	enabled := true
	endpoint := "http://otel.example.com:4317"
	protocol := "grpc"
	serviceName := "test-service"
	headers := "auth=token"

	cfg := &cfgtypes.GlobalConfig{
		Otel: &otel.Settings{
			Enabled:     &enabled,
			Endpoint:    &endpoint,
			Protocol:    &protocol,
			ServiceName: &serviceName,
			Headers:     &headers,
		},
	}

	if got := GetValue(cfg, "otel.enabled"); got != "true" {
		t.Errorf("GetValue(otel.enabled) = %q, want %q", got, "true")
	}
	if got := GetValue(cfg, "otel.endpoint"); got != endpoint {
		t.Errorf("GetValue(otel.endpoint) = %q, want %q", got, endpoint)
	}
	if got := GetValue(cfg, "otel.protocol"); got != protocol {
		t.Errorf("GetValue(otel.protocol) = %q, want %q", got, protocol)
	}
	if got := GetValue(cfg, "otel.service_name"); got != serviceName {
		t.Errorf("GetValue(otel.service_name) = %q, want %q", got, serviceName)
	}
	if got := GetValue(cfg, "otel.headers"); got != headers {
		t.Errorf("GetValue(otel.headers) = %q, want %q", got, headers)
	}
}

func TestOtelSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "otel.enabled", "true")
	if cfg.Otel == nil || cfg.Otel.Enabled == nil || *cfg.Otel.Enabled != true {
		t.Errorf("Enabled not set correctly")
	}

	SetValue(cfg, "otel.endpoint", "http://localhost:4317")
	if cfg.Otel.Endpoint == nil || *cfg.Otel.Endpoint != "http://localhost:4317" {
		t.Errorf("Endpoint not set correctly")
	}

	SetValue(cfg, "otel.protocol", "grpc")
	if cfg.Otel.Protocol == nil || *cfg.Otel.Protocol != "grpc" {
		t.Errorf("Protocol not set correctly")
	}

	SetValue(cfg, "otel.service_name", "my-service")
	if cfg.Otel.ServiceName == nil || *cfg.Otel.ServiceName != "my-service" {
		t.Errorf("ServiceName not set correctly")
	}
}

func TestOtelUnsetValue(t *testing.T) {
	enabled := true
	endpoint := "http://localhost:4317"
	cfg := &cfgtypes.GlobalConfig{
		Otel: &otel.Settings{
			Enabled:  &enabled,
			Endpoint: &endpoint,
		},
	}

	UnsetValue(cfg, "otel.enabled")
	if cfg.Otel.Enabled != nil {
		t.Errorf("Enabled should be nil after unset")
	}

	UnsetValue(cfg, "otel.endpoint")
	if cfg.Otel.Endpoint != nil {
		t.Errorf("Endpoint should be nil after unset")
	}
}

func TestOtelGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"otel.enabled", "false"},
		{"otel.endpoint", "http://host.docker.internal:4318"},
		{"otel.protocol", "http/json"},
		{"otel.service_name", "addt"},
		{"otel.headers", ""},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
