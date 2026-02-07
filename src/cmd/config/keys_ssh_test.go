package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func TestSSHKeyValidation(t *testing.T) {
	sshKeys := []string{
		"ssh.forward_keys", "ssh.forward_mode", "ssh.allowed_keys",
	}

	for _, key := range sshKeys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestSSHGetValue(t *testing.T) {
	forwardKeys := true
	cfg := &cfgtypes.GlobalConfig{
		SSH: &cfgtypes.SSHSettings{
			ForwardKeys: &forwardKeys,
			ForwardMode: "proxy",
			AllowedKeys: []string{"id_rsa", "id_ed25519"},
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"ssh.forward_keys", "true"},
		{"ssh.forward_mode", "proxy"},
		{"ssh.allowed_keys", "id_rsa,id_ed25519"},
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}

	// Test with nil SSH
	nilCfg := &cfgtypes.GlobalConfig{}
	if got := GetValue(nilCfg, "ssh.forward_keys"); got != "" {
		t.Errorf("GetValue(ssh.forward_keys) with nil SSH = %q, want empty", got)
	}
}

func TestSSHSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "ssh.forward_keys", "true")
	if cfg.SSH == nil || cfg.SSH.ForwardKeys == nil || *cfg.SSH.ForwardKeys != true {
		t.Errorf("ForwardKeys not set correctly")
	}

	SetValue(cfg, "ssh.forward_mode", "agent")
	if cfg.SSH.ForwardMode != "agent" {
		t.Errorf("ForwardMode = %q, want %q", cfg.SSH.ForwardMode, "agent")
	}

	SetValue(cfg, "ssh.allowed_keys", "id_rsa,id_ed25519")
	if len(cfg.SSH.AllowedKeys) != 2 || cfg.SSH.AllowedKeys[0] != "id_rsa" {
		t.Errorf("AllowedKeys = %v, want [id_rsa id_ed25519]", cfg.SSH.AllowedKeys)
	}
}

func TestSSHUnsetValue(t *testing.T) {
	forwardKeys := true
	cfg := &cfgtypes.GlobalConfig{
		SSH: &cfgtypes.SSHSettings{
			ForwardKeys: &forwardKeys,
			ForwardMode: "proxy",
			AllowedKeys: []string{"id_rsa"},
		},
	}

	UnsetValue(cfg, "ssh.forward_keys")
	if cfg.SSH.ForwardKeys != nil {
		t.Errorf("ForwardKeys should be nil after unset")
	}

	UnsetValue(cfg, "ssh.forward_mode")
	if cfg.SSH.ForwardMode != "" {
		t.Errorf("ForwardMode should be empty after unset")
	}

	UnsetValue(cfg, "ssh.allowed_keys")
	if cfg.SSH.AllowedKeys != nil {
		t.Errorf("AllowedKeys should be nil after unset")
	}
}

func TestSSHGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"ssh.forward_keys", "true"},
		{"ssh.forward_mode", "proxy"},
		{"ssh.allowed_keys", ""},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
