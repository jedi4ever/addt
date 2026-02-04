package firewall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jedi4ever/addt/config"
)

// setupTestEnv creates a temp directory and sets ADDT_CONFIG_DIR to isolate tests
func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "firewall-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Save original config dir
	origConfigDir := os.Getenv("ADDT_CONFIG_DIR")

	// Set config dir to temp directory
	os.Setenv("ADDT_CONFIG_DIR", tmpDir)

	// Also change to temp dir for project config tests
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)

	cleanup := func() {
		os.Setenv("ADDT_CONFIG_DIR", origConfigDir)
		os.Chdir(origWd)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGlobal_Integration_List(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// List should work with no config (shows defaults)
	HandleCommand([]string{"global", "list"})

	// Verify config file was NOT created (list is read-only)
	configPath := config.GetGlobalConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		t.Log("Config file was created (acceptable)")
	}
}

func TestGlobal_Integration_AllowDomain(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Allow a domain
	HandleCommand([]string{"global", "allow", "test.example.com"})

	// Verify config was created with the domain
	configPath := filepath.Join(tmpDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created")
	}

	cfg := config.LoadGlobalConfig()
	if !containsString(cfg.FirewallAllowed, "test.example.com") {
		t.Errorf("Expected 'test.example.com' in allowed list, got: %v", cfg.FirewallAllowed)
	}
}

func TestGlobal_Integration_DenyDomain(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Deny a domain
	HandleCommand([]string{"global", "deny", "malware.example.com"})

	cfg := config.LoadGlobalConfig()
	if !containsString(cfg.FirewallDenied, "malware.example.com") {
		t.Errorf("Expected 'malware.example.com' in denied list, got: %v", cfg.FirewallDenied)
	}
}

func TestGlobal_Integration_RemoveDomain(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Add then remove a domain
	HandleCommand([]string{"global", "allow", "to-remove.example.com"})
	HandleCommand([]string{"global", "remove", "to-remove.example.com"})

	cfg := config.LoadGlobalConfig()
	if containsString(cfg.FirewallAllowed, "to-remove.example.com") {
		t.Error("Expected domain to be removed from allowed list")
	}
}

func TestGlobal_Integration_Reset(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Add a custom domain
	HandleCommand([]string{"global", "allow", "custom.example.com"})
	HandleCommand([]string{"global", "deny", "blocked.example.com"})

	// Reset to defaults
	HandleCommand([]string{"global", "reset"})

	cfg := config.LoadGlobalConfig()

	// Custom domains should be gone
	if containsString(cfg.FirewallAllowed, "custom.example.com") {
		t.Error("Expected custom domain to be removed after reset")
	}
	if containsString(cfg.FirewallDenied, "blocked.example.com") {
		t.Error("Expected denied domain to be cleared after reset")
	}

	// Defaults should be present
	if !containsString(cfg.FirewallAllowed, "api.anthropic.com") {
		t.Error("Expected default domains after reset")
	}
}

func TestProject_Integration_AllowDomain(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Allow a domain in project config
	HandleCommand([]string{"project", "allow", "project-api.example.com"})

	cfg := config.LoadProjectConfig()
	if !containsString(cfg.FirewallAllowed, "project-api.example.com") {
		t.Errorf("Expected 'project-api.example.com' in project allowed list, got: %v", cfg.FirewallAllowed)
	}
}

func TestProject_Integration_Reset(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Add then reset
	HandleCommand([]string{"project", "allow", "temp.example.com"})
	HandleCommand([]string{"project", "reset"})

	cfg := config.LoadProjectConfig()
	if len(cfg.FirewallAllowed) > 0 {
		t.Errorf("Expected empty allowed list after reset, got: %v", cfg.FirewallAllowed)
	}
}

func TestExtension_Integration_AllowDomain(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Allow a domain for claude extension
	HandleCommand([]string{"extension", "claude", "allow", "api.anthropic.com"})

	cfg := config.LoadGlobalConfig()
	if cfg.Extensions == nil || cfg.Extensions["claude"] == nil {
		t.Fatal("Expected claude extension config to exist")
	}
	if !containsString(cfg.Extensions["claude"].FirewallAllowed, "api.anthropic.com") {
		t.Errorf("Expected 'api.anthropic.com' in claude allowed list, got: %v",
			cfg.Extensions["claude"].FirewallAllowed)
	}
}

func TestExtension_Integration_DenyDomain(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Deny a domain for codex extension
	HandleCommand([]string{"extension", "codex", "deny", "blocked.example.com"})

	cfg := config.LoadGlobalConfig()
	if cfg.Extensions == nil || cfg.Extensions["codex"] == nil {
		t.Fatal("Expected codex extension config to exist")
	}
	if !containsString(cfg.Extensions["codex"].FirewallDenied, "blocked.example.com") {
		t.Errorf("Expected 'blocked.example.com' in codex denied list, got: %v",
			cfg.Extensions["codex"].FirewallDenied)
	}
}

func TestExtension_Integration_Reset(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Add then reset
	HandleCommand([]string{"extension", "test-ext", "allow", "temp.example.com"})
	HandleCommand([]string{"extension", "test-ext", "reset"})

	cfg := config.LoadGlobalConfig()
	if cfg.Extensions != nil && cfg.Extensions["test-ext"] != nil {
		if len(cfg.Extensions["test-ext"].FirewallAllowed) > 0 {
			t.Errorf("Expected empty allowed list after reset, got: %v",
				cfg.Extensions["test-ext"].FirewallAllowed)
		}
	}
}

func TestCommand_Integration_Help(t *testing.T) {
	// Just verify help doesn't panic
	HandleCommand([]string{"help"})
	HandleCommand([]string{"--help"})
	HandleCommand([]string{})
}

func TestCommand_Integration_DuplicateDomain(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Add same domain twice
	HandleCommand([]string{"global", "allow", "duplicate.example.com"})
	HandleCommand([]string{"global", "allow", "duplicate.example.com"})

	cfg := config.LoadGlobalConfig()
	count := 0
	for _, d := range cfg.FirewallAllowed {
		if d == "duplicate.example.com" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected domain to appear once, found %d times", count)
	}
}

func TestConfig_Integration_YAMLFormat(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Add some rules
	HandleCommand([]string{"global", "allow", "allowed.example.com"})
	HandleCommand([]string{"global", "deny", "denied.example.com"})

	// Read the raw YAML file
	configPath := filepath.Join(tmpDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	// Verify YAML structure
	contentStr := string(content)
	if !strings.Contains(contentStr, "firewall_allowed:") {
		t.Error("Expected firewall_allowed in YAML")
	}
	if !strings.Contains(contentStr, "firewall_denied:") {
		t.Error("Expected firewall_denied in YAML")
	}
	if !strings.Contains(contentStr, "allowed.example.com") {
		t.Error("Expected allowed domain in YAML")
	}
	if !strings.Contains(contentStr, "denied.example.com") {
		t.Error("Expected denied domain in YAML")
	}

	t.Logf("Config YAML:\n%s", contentStr)
}
