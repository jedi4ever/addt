package firewall

import (
	"testing"

	"github.com/jedi4ever/addt/config"
)

func TestCheckDomain_LayeredOverride(t *testing.T) {
	tests := []struct {
		name        string
		domain      string
		cfg         *config.Config
		wantAllowed bool
		wantLayer   string
	}{
		{
			name:   "default allowed",
			domain: "api.anthropic.com",
			cfg: &config.Config{
				GlobalFirewallAllowed:    nil,
				GlobalFirewallDenied:     nil,
				ProjectFirewallAllowed:   nil,
				ProjectFirewallDenied:    nil,
				ExtensionFirewallAllowed: nil,
				ExtensionFirewallDenied:  nil,
			},
			wantAllowed: true,
			wantLayer:   "defaults",
		},
		{
			name:   "global denies default",
			domain: "registry.npmjs.org",
			cfg: &config.Config{
				GlobalFirewallDenied: []string{"registry.npmjs.org"},
			},
			wantAllowed: false,
			wantLayer:   "global",
		},
		{
			name:   "project re-allows what global denied",
			domain: "registry.npmjs.org",
			cfg: &config.Config{
				GlobalFirewallDenied:   []string{"registry.npmjs.org"},
				ProjectFirewallAllowed: []string{"registry.npmjs.org"},
			},
			wantAllowed: true,
			wantLayer:   "project",
		},
		{
			name:   "project denies what global allowed",
			domain: "custom.example.com",
			cfg: &config.Config{
				GlobalFirewallAllowed: []string{"custom.example.com"},
				ProjectFirewallDenied: []string{"custom.example.com"},
			},
			wantAllowed: false,
			wantLayer:   "project",
		},
		{
			name:   "extension allows, global denies, project wins",
			domain: "api.openai.com",
			cfg: &config.Config{
				ExtensionFirewallAllowed: []string{"api.openai.com"},
				GlobalFirewallDenied:     []string{"api.openai.com"},
				ProjectFirewallAllowed:   []string{"api.openai.com"},
			},
			wantAllowed: true,
			wantLayer:   "project",
		},
		{
			name:   "not in any list - no match",
			domain: "unknown.example.com",
			cfg: &config.Config{
				GlobalFirewallAllowed: []string{"other.example.com"},
			},
			wantAllowed: false,
			wantLayer:   "none",
		},
		{
			name:   "global allows custom domain",
			domain: "custom.api.com",
			cfg: &config.Config{
				GlobalFirewallAllowed: []string{"custom.api.com"},
			},
			wantAllowed: true,
			wantLayer:   "global",
		},
		{
			name:   "extension allows, not overridden",
			domain: "ext.api.com",
			cfg: &config.Config{
				ExtensionFirewallAllowed: []string{"ext.api.com"},
			},
			wantAllowed: true,
			wantLayer:   "extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, layer := CheckDomain(tt.domain, tt.cfg, "")
			if allowed != tt.wantAllowed {
				t.Errorf("CheckDomain() allowed = %v, want %v", allowed, tt.wantAllowed)
			}
			if layer != tt.wantLayer {
				t.Errorf("CheckDomain() layer = %v, want %v", layer, tt.wantLayer)
			}
		})
	}
}
