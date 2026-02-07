package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/config/security"
)

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
