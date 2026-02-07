package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func TestContainerKeyValidation(t *testing.T) {
	keys := []string{"container.cpus", "container.memory"}
	for _, key := range keys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestContainerGetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{
		Container: &cfgtypes.ContainerSettings{
			CPUs:   "4",
			Memory: "8g",
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"container.cpus", "4"},
		{"container.memory", "8g"},
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}

	// Test with nil Container
	nilCfg := &cfgtypes.GlobalConfig{}
	if got := GetValue(nilCfg, "container.cpus"); got != "" {
		t.Errorf("GetValue(container.cpus) with nil Container = %q, want empty", got)
	}
}

func TestContainerSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "container.cpus", "2")
	if cfg.Container == nil || cfg.Container.CPUs != "2" {
		t.Errorf("CPUs not set correctly")
	}

	SetValue(cfg, "container.memory", "4g")
	if cfg.Container.Memory != "4g" {
		t.Errorf("Memory = %q, want %q", cfg.Container.Memory, "4g")
	}
}

func TestContainerUnsetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{
		Container: &cfgtypes.ContainerSettings{
			CPUs:   "4",
			Memory: "8g",
		},
	}

	UnsetValue(cfg, "container.cpus")
	if cfg.Container.CPUs != "" {
		t.Errorf("CPUs should be empty after unset")
	}

	UnsetValue(cfg, "container.memory")
	if cfg.Container.Memory != "" {
		t.Errorf("Memory should be empty after unset")
	}
}

func TestContainerGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"container.cpus", "2"},
		{"container.memory", "4g"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
