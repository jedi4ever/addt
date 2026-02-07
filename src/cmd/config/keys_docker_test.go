package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func TestDockerKeyValidation(t *testing.T) {
	dockerKeys := []string{
		"docker.cpus", "docker.memory",
		"docker.dind.enable", "docker.dind.mode",
	}

	for _, key := range dockerKeys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestDockerGetValue(t *testing.T) {
	dindEnable := true
	cfg := &cfgtypes.GlobalConfig{
		Docker: &cfgtypes.DockerSettings{
			CPUs:   "4",
			Memory: "8g",
			Dind: &cfgtypes.DindSettings{
				Enable: &dindEnable,
				Mode:   "isolated",
			},
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"docker.cpus", "4"},
		{"docker.memory", "8g"},
		{"docker.dind.enable", "true"},
		{"docker.dind.mode", "isolated"},
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}

	// Test with nil Docker
	nilCfg := &cfgtypes.GlobalConfig{}
	if got := GetValue(nilCfg, "docker.cpus"); got != "" {
		t.Errorf("GetValue(docker.cpus) with nil Docker = %q, want empty", got)
	}
}

func TestDockerSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "docker.cpus", "2")
	if cfg.Docker == nil || cfg.Docker.CPUs != "2" {
		t.Errorf("CPUs not set correctly")
	}

	SetValue(cfg, "docker.memory", "4g")
	if cfg.Docker.Memory != "4g" {
		t.Errorf("Memory = %q, want %q", cfg.Docker.Memory, "4g")
	}

	SetValue(cfg, "docker.dind.enable", "true")
	if cfg.Docker.Dind == nil || cfg.Docker.Dind.Enable == nil || *cfg.Docker.Dind.Enable != true {
		t.Errorf("Dind.Enable not set correctly")
	}

	SetValue(cfg, "docker.dind.mode", "host")
	if cfg.Docker.Dind.Mode != "host" {
		t.Errorf("Dind.Mode = %q, want %q", cfg.Docker.Dind.Mode, "host")
	}
}

func TestDockerUnsetValue(t *testing.T) {
	dindEnable := true
	cfg := &cfgtypes.GlobalConfig{
		Docker: &cfgtypes.DockerSettings{
			CPUs:   "4",
			Memory: "8g",
			Dind: &cfgtypes.DindSettings{
				Enable: &dindEnable,
				Mode:   "isolated",
			},
		},
	}

	UnsetValue(cfg, "docker.cpus")
	if cfg.Docker.CPUs != "" {
		t.Errorf("CPUs should be empty after unset")
	}

	UnsetValue(cfg, "docker.memory")
	if cfg.Docker.Memory != "" {
		t.Errorf("Memory should be empty after unset")
	}

	UnsetValue(cfg, "docker.dind.enable")
	if cfg.Docker.Dind.Enable != nil {
		t.Errorf("Dind.Enable should be nil after unset")
	}

	UnsetValue(cfg, "docker.dind.mode")
	if cfg.Docker.Dind.Mode != "" {
		t.Errorf("Dind.Mode should be empty after unset")
	}
}

func TestDockerGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"docker.cpus", ""},
		{"docker.memory", ""},
		{"docker.dind.enable", "false"},
		{"docker.dind.mode", "isolated"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
