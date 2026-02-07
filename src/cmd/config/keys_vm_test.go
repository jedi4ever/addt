package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func TestVmKeyValidation(t *testing.T) {
	keys := []string{"vm.cpus", "vm.memory"}
	for _, key := range keys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestVmGetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{
		Vm: &cfgtypes.VmSettings{
			CPUs:   "8",
			Memory: "16384",
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"vm.cpus", "8"},
		{"vm.memory", "16384"},
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}

	// Test with nil Vm
	nilCfg := &cfgtypes.GlobalConfig{}
	if got := GetValue(nilCfg, "vm.cpus"); got != "" {
		t.Errorf("GetValue(vm.cpus) with nil Vm = %q, want empty", got)
	}
}

func TestVmSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "vm.cpus", "4")
	if cfg.Vm == nil || cfg.Vm.CPUs != "4" {
		t.Errorf("CPUs not set correctly")
	}

	SetValue(cfg, "vm.memory", "8192")
	if cfg.Vm.Memory != "8192" {
		t.Errorf("Memory = %q, want %q", cfg.Vm.Memory, "8192")
	}
}

func TestVmUnsetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{
		Vm: &cfgtypes.VmSettings{
			CPUs:   "4",
			Memory: "8192",
		},
	}

	UnsetValue(cfg, "vm.cpus")
	if cfg.Vm.CPUs != "" {
		t.Errorf("CPUs should be empty after unset")
	}

	UnsetValue(cfg, "vm.memory")
	if cfg.Vm.Memory != "" {
		t.Errorf("Memory should be empty after unset")
	}
}

func TestVmGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"vm.cpus", "4"},
		{"vm.memory", "8192"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
