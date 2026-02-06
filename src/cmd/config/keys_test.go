package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/config/otel"
	"github.com/jedi4ever/addt/config/security"
)

func TestKeyValidation(t *testing.T) {
	validKeys := []string{
		"docker_cpus", "docker_memory", "dind", "dind_mode",
		"firewall", "firewall_mode", "node_version", "go_version",
		"persistent", "workdir", "workdir_automount",
	}

	for _, key := range validKeys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}

	invalidKeys := []string{"invalid", "foo", "bar", "version"}
	for _, key := range invalidKeys {
		if IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = true, want false", key)
		}
	}
}

func TestExtensionKeyValidation(t *testing.T) {
	validKeys := []string{"version", "automount"}
	for _, key := range validKeys {
		if !IsValidExtensionKey(key) {
			t.Errorf("IsValidExtensionKey(%q) = false, want true", key)
		}
	}

	invalidKeys := []string{"invalid", "foo", "node_version"}
	for _, key := range invalidKeys {
		if IsValidExtensionKey(key) {
			t.Errorf("IsValidExtensionKey(%q) = true, want false", key)
		}
	}
}

func TestGetValue(t *testing.T) {
	persistent := true
	portStart := 35000
	cfg := &cfgtypes.GlobalConfig{
		NodeVersion:    "20",
		DockerCPUs:     "4",
		Persistent:     &persistent,
		PortRangeStart: &portStart,
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"node_version", "20"},
		{"docker_cpus", "4"},
		{"persistent", "true"},
		{"port_range_start", "35000"},
		{"go_version", ""}, // not set
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "node_version", "18")
	if cfg.NodeVersion != "18" {
		t.Errorf("NodeVersion = %q, want %q", cfg.NodeVersion, "18")
	}

	SetValue(cfg, "persistent", "true")
	if cfg.Persistent == nil || *cfg.Persistent != true {
		t.Errorf("Persistent = %v, want true", cfg.Persistent)
	}

	SetValue(cfg, "port_range_start", "40000")
	if cfg.PortRangeStart == nil || *cfg.PortRangeStart != 40000 {
		t.Errorf("PortRangeStart = %v, want 40000", cfg.PortRangeStart)
	}
}

func TestUnsetValue(t *testing.T) {
	persistent := true
	cfg := &cfgtypes.GlobalConfig{
		NodeVersion: "20",
		Persistent:  &persistent,
	}

	UnsetValue(cfg, "node_version")
	if cfg.NodeVersion != "" {
		t.Errorf("NodeVersion = %q, want empty", cfg.NodeVersion)
	}

	UnsetValue(cfg, "persistent")
	if cfg.Persistent != nil {
		t.Errorf("Persistent = %v, want nil", cfg.Persistent)
	}
}

func TestGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"node_version", "22"},
		{"firewall", "false"},
		{"firewall_mode", "strict"},
		{"persistent", "false"},
		{"workdir_automount", "true"},
		{"port_range_start", "30000"},
		{"ssh_forward", "agent"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestSecurityKeyValidation(t *testing.T) {
	validKeys := []string{
		"security.pids_limit",
		"security.isolate_secrets",
		"security.network_mode",
		"security.cap_drop",
		"security.cap_add",
	}

	for _, key := range validKeys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestSecurityGetValue(t *testing.T) {
	pidsLimit := 100
	isolateSecrets := true
	cfg := &cfgtypes.GlobalConfig{
		Security: &security.Settings{
			PidsLimit:      &pidsLimit,
			IsolateSecrets: &isolateSecrets,
			NetworkMode:    "none",
			CapDrop:        []string{"ALL"},
			CapAdd:         []string{"CHOWN", "SETUID"},
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"security.pids_limit", "100"},
		{"security.isolate_secrets", "true"},
		{"security.network_mode", "none"},
		{"security.cap_drop", "ALL"},
		{"security.cap_add", "CHOWN,SETUID"},
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestSecuritySetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "security.pids_limit", "150")
	if cfg.Security == nil || cfg.Security.PidsLimit == nil || *cfg.Security.PidsLimit != 150 {
		t.Errorf("PidsLimit not set correctly")
	}

	SetValue(cfg, "security.isolate_secrets", "true")
	if cfg.Security.IsolateSecrets == nil || *cfg.Security.IsolateSecrets != true {
		t.Errorf("IsolateSecrets not set correctly")
	}

	SetValue(cfg, "security.network_mode", "none")
	if cfg.Security.NetworkMode != "none" {
		t.Errorf("NetworkMode = %q, want %q", cfg.Security.NetworkMode, "none")
	}

	SetValue(cfg, "security.cap_drop", "ALL,NET_RAW")
	if len(cfg.Security.CapDrop) != 2 || cfg.Security.CapDrop[0] != "ALL" {
		t.Errorf("CapDrop = %v, want [ALL, NET_RAW]", cfg.Security.CapDrop)
	}
}

func TestSecurityUnsetValue(t *testing.T) {
	pidsLimit := 100
	isolateSecrets := true
	cfg := &cfgtypes.GlobalConfig{
		Security: &security.Settings{
			PidsLimit:      &pidsLimit,
			IsolateSecrets: &isolateSecrets,
			NetworkMode:    "none",
		},
	}

	UnsetValue(cfg, "security.pids_limit")
	if cfg.Security.PidsLimit != nil {
		t.Errorf("PidsLimit should be nil after unset")
	}

	UnsetValue(cfg, "security.isolate_secrets")
	if cfg.Security.IsolateSecrets != nil {
		t.Errorf("IsolateSecrets should be nil after unset")
	}

	UnsetValue(cfg, "security.network_mode")
	if cfg.Security.NetworkMode != "" {
		t.Errorf("NetworkMode should be empty after unset")
	}
}

func TestSecurityGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"security.pids_limit", "200"},
		{"security.no_new_privileges", "true"},
		{"security.isolate_secrets", "false"},
		{"security.cap_drop", "ALL"},
		{"security.cap_add", "CHOWN,SETUID,SETGID"},
		{"security.ulimit_nofile", "4096:8192"},
		{"security.time_limit", "0"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

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
