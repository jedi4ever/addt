package config

import (
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

func TestGitHubKeyValidation(t *testing.T) {
	githubKeys := []string{
		"github.forward_token", "github.token_source",
	}

	for _, key := range githubKeys {
		if !IsValidKey(key) {
			t.Errorf("IsValidKey(%q) = false, want true", key)
		}
	}
}

func TestGitHubGetValue(t *testing.T) {
	forwardToken := true
	cfg := &cfgtypes.GlobalConfig{
		GitHub: &cfgtypes.GitHubSettings{
			ForwardToken: &forwardToken,
			TokenSource:  "gh_auth",
		},
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"github.forward_token", "true"},
		{"github.token_source", "gh_auth"},
	}

	for _, tt := range tests {
		got := GetValue(cfg, tt.key)
		if got != tt.expected {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}

	// Test with nil GitHub
	nilCfg := &cfgtypes.GlobalConfig{}
	if got := GetValue(nilCfg, "github.forward_token"); got != "" {
		t.Errorf("GetValue(github.forward_token) with nil GitHub = %q, want empty", got)
	}
}

func TestGitHubSetValue(t *testing.T) {
	cfg := &cfgtypes.GlobalConfig{}

	SetValue(cfg, "github.forward_token", "false")
	if cfg.GitHub == nil || cfg.GitHub.ForwardToken == nil || *cfg.GitHub.ForwardToken != false {
		t.Errorf("ForwardToken not set correctly")
	}

	SetValue(cfg, "github.token_source", "gh_auth")
	if cfg.GitHub.TokenSource != "gh_auth" {
		t.Errorf("TokenSource = %q, want %q", cfg.GitHub.TokenSource, "gh_auth")
	}
}

func TestGitHubUnsetValue(t *testing.T) {
	forwardToken := true
	cfg := &cfgtypes.GlobalConfig{
		GitHub: &cfgtypes.GitHubSettings{
			ForwardToken: &forwardToken,
			TokenSource:  "gh_auth",
		},
	}

	UnsetValue(cfg, "github.forward_token")
	if cfg.GitHub.ForwardToken != nil {
		t.Errorf("ForwardToken should be nil after unset")
	}

	UnsetValue(cfg, "github.token_source")
	if cfg.GitHub.TokenSource != "" {
		t.Errorf("TokenSource should be empty after unset")
	}
}

func TestGitHubGetDefaultValue(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"github.forward_token", "true"},
		{"github.token_source", "gh_auth"},
	}

	for _, tt := range tests {
		got := GetDefaultValue(tt.key)
		if got != tt.expected {
			t.Errorf("GetDefaultValue(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}
