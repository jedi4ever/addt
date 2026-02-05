package podman

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jedi4ever/addt/provider"
)

func TestBuildPastaOptions(t *testing.T) {
	tests := []struct {
		name         string
		firewallMode string
		homeDir      string
		expected     string
	}{
		{
			name:         "off mode returns empty",
			firewallMode: "off",
			homeDir:      "/home/test",
			expected:     "",
		},
		{
			name:         "disabled mode returns empty",
			firewallMode: "disabled",
			homeDir:      "/home/test",
			expected:     "",
		},
		{
			name:         "permissive mode returns pasta",
			firewallMode: "permissive",
			homeDir:      "/home/test",
			expected:     "pasta",
		},
		{
			name:         "strict mode returns pasta with options",
			firewallMode: "strict",
			homeDir:      "/home/test",
			expected:     "pasta:-T 443:-T 80:-T 22:-T 53:-T 5432:-T 3306:-T 6379",
		},
		{
			name:         "empty mode defaults to strict",
			firewallMode: "",
			homeDir:      "/home/test",
			expected:     "pasta:-T 443:-T 80:-T 22:-T 53:-T 5432:-T 3306:-T 6379",
		},
		{
			name:         "unknown mode returns pasta",
			firewallMode: "unknown",
			homeDir:      "/home/test",
			expected:     "pasta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPastaOptions(tt.firewallMode, tt.homeDir)
			if result != tt.expected {
				t.Errorf("buildPastaOptions(%q, %q) = %q, want %q",
					tt.firewallMode, tt.homeDir, result, tt.expected)
			}
		})
	}
}

func TestGetAllowedPorts(t *testing.T) {
	// Test with default ports (no config file)
	ports := getAllowedPorts("/nonexistent")
	expected := []string{"443", "80", "22", "53", "5432", "3306", "6379"}

	if len(ports) != len(expected) {
		t.Errorf("getAllowedPorts() returned %d ports, want %d", len(ports), len(expected))
	}

	for i, p := range ports {
		if p != expected[i] {
			t.Errorf("getAllowedPorts()[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestGetAllowedPortsWithConfigFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "addt-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create firewall config directory
	firewallDir := filepath.Join(tmpDir, ".addt", "firewall")
	if err := os.MkdirAll(firewallDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write custom ports file
	portsFile := filepath.Join(firewallDir, "allowed-ports.txt")
	content := "8080\n# Comment\n8443\n\n9000\n"
	if err := os.WriteFile(portsFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Test reading custom ports
	ports := getAllowedPorts(tmpDir)
	expected := []string{"8080", "8443", "9000"}

	if len(ports) != len(expected) {
		t.Errorf("getAllowedPorts() returned %d ports, want %d", len(ports), len(expected))
	}

	for i, p := range ports {
		if p != expected[i] {
			t.Errorf("getAllowedPorts()[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestHandleFirewallConfig_Disabled(t *testing.T) {
	p := &PodmanProvider{
		config: &provider.Config{
			FirewallEnabled: false,
		},
	}

	args := p.HandleFirewallConfig("/home/test", "addt")
	if len(args) != 0 {
		t.Errorf("HandleFirewallConfig() with firewall disabled returned %d args, want 0", len(args))
	}
}

func TestHandleFirewallConfig_Enabled(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "addt-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	p := &PodmanProvider{
		config: &provider.Config{
			FirewallEnabled: true,
			FirewallMode:    "permissive",
		},
	}

	args := p.HandleFirewallConfig(tmpDir, "addt")

	// Should have --network, pasta options, firewall config mount, and NET_ADMIN
	hasNetwork := false
	hasNetAdmin := false
	hasVolume := false

	for i, arg := range args {
		if arg == "--network" && i+1 < len(args) && args[i+1] == "pasta" {
			hasNetwork = true
		}
		if arg == "--cap-add" && i+1 < len(args) && args[i+1] == "NET_ADMIN" {
			hasNetAdmin = true
		}
		if arg == "-v" {
			hasVolume = true
		}
	}

	if !hasNetwork {
		t.Error("HandleFirewallConfig() missing --network pasta")
	}
	if !hasNetAdmin {
		t.Error("HandleFirewallConfig() missing --cap-add NET_ADMIN")
	}
	if !hasVolume {
		t.Error("HandleFirewallConfig() missing volume mount for firewall config")
	}
}

func TestEnsureFirewallConfigDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "addt-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testDir := filepath.Join(tmpDir, "firewall")

	// Directory should not exist yet
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatal("test directory should not exist initially")
	}

	// Create it
	if err := ensureFirewallConfigDir(testDir); err != nil {
		t.Errorf("ensureFirewallConfigDir() error = %v", err)
	}

	// Directory should exist now
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("ensureFirewallConfigDir() did not create directory")
	}

	// Calling again should succeed (idempotent)
	if err := ensureFirewallConfigDir(testDir); err != nil {
		t.Errorf("ensureFirewallConfigDir() second call error = %v", err)
	}
}

func TestWriteAllowedDomains(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "addt-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	domains := []string{"github.com", "api.anthropic.com", "registry.npmjs.org"}

	if err := WriteAllowedDomains(tmpDir, domains); err != nil {
		t.Errorf("WriteAllowedDomains() error = %v", err)
	}

	// Verify file was created
	filename := GetAllowedDomainsFile(tmpDir)
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("Failed to read domains file: %v", err)
	}

	expected := "github.com\napi.anthropic.com\nregistry.npmjs.org\n"
	if string(content) != expected {
		t.Errorf("WriteAllowedDomains() wrote %q, want %q", string(content), expected)
	}
}

func TestGetAllowedDomainsFile(t *testing.T) {
	result := GetAllowedDomainsFile("/home/test")
	expected := "/home/test/.addt/firewall/allowed-domains.txt"
	if result != expected {
		t.Errorf("GetAllowedDomainsFile() = %q, want %q", result, expected)
	}
}
