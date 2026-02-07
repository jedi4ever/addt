package config

import (
	"strings"
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func TestKeyValidation(t *testing.T) {
	validKeys := []string{
		"container.cpus", "container.memory",
		"docker.dind.enable", "docker.dind.mode",
		"env_file_load", "env_file",
		"firewall.enabled", "firewall.mode",
		"github.forward_token", "github.token_source",
		"gpg.forward", "gpg.allowed_key_ids",
		"log.enabled", "log.output", "log.file", "log.dir", "log.level", "log.modules",
		"log.rotate", "log.max_size", "log.max_files",
		"node_version", "go_version",
		"persistent", "ports.forward", "ports.expose", "ports.inject_system_prompt", "ports.range_start",
		"vm.cpus", "vm.memory",
		"workdir.path", "workdir.automount", "workdir.readonly",
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
	// Static keys are valid for any extension
	validKeys := []string{"version", "automount"}
	for _, key := range validKeys {
		if !IsValidExtensionKey(key, "claude") {
			t.Errorf("IsValidExtensionKey(%q, \"claude\") = false, want true", key)
		}
	}

	invalidKeys := []string{"invalid", "foo", "node_version"}
	for _, key := range invalidKeys {
		if IsValidExtensionKey(key, "claude") {
			t.Errorf("IsValidExtensionKey(%q, \"claude\") = true, want false", key)
		}
	}
}

func TestExtensionFlagKeyValidation(t *testing.T) {
	// "yolo" is a flag key defined in claude's config.yaml
	if !IsValidExtensionKey("yolo", "claude") {
		t.Error("IsValidExtensionKey(\"yolo\", \"claude\") = false, want true")
	}

	// "yolo" should NOT be valid for an extension that doesn't define it
	if IsValidExtensionKey("yolo", "nonexistent") {
		t.Error("IsValidExtensionKey(\"yolo\", \"nonexistent\") = true, want false")
	}

	// IsFlagKey should identify flag keys
	if !IsFlagKey("yolo", "claude") {
		t.Error("IsFlagKey(\"yolo\", \"claude\") = false, want true")
	}

	// Static keys should NOT be flag keys
	if IsFlagKey("version", "claude") {
		t.Error("IsFlagKey(\"version\", \"claude\") = true, want false")
	}
}

func TestGetExtensionFlagKeys(t *testing.T) {
	keys := GetExtensionFlagKeys("claude")
	if len(keys) == 0 {
		t.Fatal("GetExtensionFlagKeys(\"claude\") returned no keys, expected at least 'yolo'")
	}

	found := false
	for _, k := range keys {
		if k.Key == "yolo" {
			found = true
			if k.Type != "bool" {
				t.Errorf("yolo key type = %q, want \"bool\"", k.Type)
			}
			if k.EnvVar != "ADDT_EXTENSION_CLAUDE_YOLO" {
				t.Errorf("yolo key EnvVar = %q, want \"ADDT_EXTENSION_CLAUDE_YOLO\"", k.EnvVar)
			}
		}
	}
	if !found {
		t.Error("GetExtensionFlagKeys(\"claude\") missing 'yolo' key")
	}
}

func TestAvailableExtensionKeyNames(t *testing.T) {
	names := AvailableExtensionKeyNames("claude")
	if names == "" {
		t.Fatal("AvailableExtensionKeyNames returned empty string")
	}
	// Should contain both static and flag keys
	if !strings.Contains(names, "version") {
		t.Errorf("AvailableExtensionKeyNames missing 'version': %s", names)
	}
	if !strings.Contains(names, "yolo") {
		t.Errorf("AvailableExtensionKeyNames missing 'yolo': %s", names)
	}
}

func TestGetValue(t *testing.T) {
	persistent := true
	portStart := 35000
	portsForward := true
	portsInjectSystemPrompt := true
	cfg := &cfgtypes.GlobalConfig{
		NodeVersion: "20",
		Container: &cfgtypes.ContainerSettings{
			CPUs: "4",
		},
		Persistent: &persistent,
		Ports: &cfgtypes.PortsSettings{
			Forward:            &portsForward,
			Expose:             []string{"3000", "8080"},
			RangeStart:         &portStart,
			InjectSystemPrompt: &portsInjectSystemPrompt,
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"node_version", "20"},
		{"container.cpus", "4"},
		{"persistent", "true"},
		{"ports.forward", "true"},
		{"ports.expose", "3000,8080"},
		{"ports.inject_system_prompt", "true"},
		{"ports.range_start", "35000"},
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
		{"env_file_load", "true"},
		{"env_file", ".env"},
		{"node_version", "22"},
		{"firewall.enabled", "false"},
		{"firewall.mode", "strict"},
		{"persistent", "false"},
		{"workdir.automount", "true"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestEnvFileGetValue(t *testing.T) {
	envFileLoad := false
	cfg := &cfgtypes.GlobalConfig{
		EnvFileLoad: &envFileLoad,
		EnvFile:     "/path/to/.env",
	}

	if got := GetValue(cfg, "env_file_load"); got != "false" {
		t.Errorf("GetValue(env_file_load) = %q, want %q", got, "false")
	}
	if got := GetValue(cfg, "env_file"); got != "/path/to/.env" {
		t.Errorf("GetValue(env_file) = %q, want %q", got, "/path/to/.env")
	}

	// nil EnvFileLoad returns empty
	nilCfg := &cfgtypes.GlobalConfig{}
	if got := GetValue(nilCfg, "env_file_load"); got != "" {
		t.Errorf("GetValue(env_file_load) with nil = %q, want empty", got)
	}
}

func TestEnvFileSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "env_file_load", "false")
	if cfg.EnvFileLoad == nil || *cfg.EnvFileLoad != false {
		t.Errorf("EnvFileLoad not set correctly")
	}

	SetValue(cfg, "env_file", "custom.env")
	if cfg.EnvFile != "custom.env" {
		t.Errorf("EnvFile = %q, want %q", cfg.EnvFile, "custom.env")
	}
}

func TestEnvFileUnsetValue(t *testing.T) {
	envFileLoad := true
	cfg := &cfgtypes.GlobalConfig{
		EnvFileLoad: &envFileLoad,
		EnvFile:     "custom.env",
	}

	UnsetValue(cfg, "env_file_load")
	if cfg.EnvFileLoad != nil {
		t.Errorf("EnvFileLoad should be nil after unset")
	}

	UnsetValue(cfg, "env_file")
	if cfg.EnvFile != "" {
		t.Errorf("EnvFile should be empty after unset")
	}
}
