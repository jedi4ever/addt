package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func TestPortsKeyValidation(t *testing.T) {
	portsKeys := []string{
		"ports.forward", "ports.expose",
		"ports.inject_system_prompt", "ports.range_start",
	}

	for _, key := range portsKeys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestPortsGetValue(t *testing.T) {
	portStart := 35000
	portsForward := true
	portsInjectSystemPrompt := true
	cfg := &cfgtypes.GlobalConfig{
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
		{"ports.forward", "true"},
		{"ports.expose", "3000,8080"},
		{"ports.inject_system_prompt", "true"},
		{"ports.range_start", "35000"},
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestPortsSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "ports.expose", "3000, 8080")
	if cfg.Ports == nil || len(cfg.Ports.Expose) != 2 || cfg.Ports.Expose[0] != "3000" || cfg.Ports.Expose[1] != "8080" {
		t.Errorf("Ports.Expose = %v, want [3000 8080]", cfg.Ports)
	}

	SetValue(cfg, "ports.range_start", "40000")
	if cfg.Ports == nil || cfg.Ports.RangeStart == nil || *cfg.Ports.RangeStart != 40000 {
		t.Errorf("Ports.RangeStart = %v, want 40000", cfg.Ports)
	}

	SetValue(cfg, "ports.forward", "true")
	if cfg.Ports == nil || cfg.Ports.Forward == nil || *cfg.Ports.Forward != true {
		t.Errorf("Ports.Forward not set correctly")
	}

	SetValue(cfg, "ports.inject_system_prompt", "false")
	if cfg.Ports == nil || cfg.Ports.InjectSystemPrompt == nil || *cfg.Ports.InjectSystemPrompt != false {
		t.Errorf("Ports.InjectSystemPrompt not set correctly")
	}
}

func TestPortsUnsetValue(t *testing.T) {
	portsForward := true
	portsInjectSystemPrompt := true
	cfg := &cfgtypes.GlobalConfig{
		Ports: &cfgtypes.PortsSettings{
			Forward:            &portsForward,
			Expose:             []string{"3000", "8080"},
			InjectSystemPrompt: &portsInjectSystemPrompt,
		},
	}

	UnsetValue(cfg, "ports.expose")
	if cfg.Ports.Expose != nil {
		t.Errorf("Ports.Expose = %v, want nil", cfg.Ports.Expose)
	}

	UnsetValue(cfg, "ports.forward")
	if cfg.Ports.Forward != nil {
		t.Errorf("Ports.Forward = %v, want nil", cfg.Ports.Forward)
	}

	UnsetValue(cfg, "ports.inject_system_prompt")
	if cfg.Ports.InjectSystemPrompt != nil {
		t.Errorf("Ports.InjectSystemPrompt = %v, want nil", cfg.Ports.InjectSystemPrompt)
	}
}

func TestPortsGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"ports.forward", "true"},
		{"ports.expose", ""},
		{"ports.inject_system_prompt", "true"},
		{"ports.range_start", "30000"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
