package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// setupTestEnv creates temporary directories for testing config precedence
// Returns cleanup function to restore original state
func setupTestEnv(t *testing.T) (globalDir, projectDir string, cleanup func()) {
	t.Helper()

	// Save original env vars
	origConfigDir := os.Getenv("ADDT_CONFIG_DIR")
	origNodeVersion := os.Getenv("ADDT_NODE_VERSION")
	origGoVersion := os.Getenv("ADDT_GO_VERSION")
	origPersistent := os.Getenv("ADDT_PERSISTENT")
	origFirewallMode := os.Getenv("ADDT_FIREWALL_MODE")
	origCwd, _ := os.Getwd()

	// Create temp directories
	globalDir = t.TempDir()
	projectDir = t.TempDir()

	// Set ADDT_CONFIG_DIR to isolate from real home directory
	os.Setenv("ADDT_CONFIG_DIR", globalDir)

	// Clear env vars that might interfere
	os.Unsetenv("ADDT_NODE_VERSION")
	os.Unsetenv("ADDT_GO_VERSION")
	os.Unsetenv("ADDT_PERSISTENT")
	os.Unsetenv("ADDT_FIREWALL_MODE")

	// Change to project directory
	os.Chdir(projectDir)

	cleanup = func() {
		// Restore env vars
		if origConfigDir != "" {
			os.Setenv("ADDT_CONFIG_DIR", origConfigDir)
		} else {
			os.Unsetenv("ADDT_CONFIG_DIR")
		}
		if origNodeVersion != "" {
			os.Setenv("ADDT_NODE_VERSION", origNodeVersion)
		}
		if origGoVersion != "" {
			os.Setenv("ADDT_GO_VERSION", origGoVersion)
		}
		if origPersistent != "" {
			os.Setenv("ADDT_PERSISTENT", origPersistent)
		}
		if origFirewallMode != "" {
			os.Setenv("ADDT_FIREWALL_MODE", origFirewallMode)
		}
		os.Chdir(origCwd)
	}

	return globalDir, projectDir, cleanup
}

// writeGlobalConfig writes a GlobalConfig to the global config file
func writeGlobalConfig(t *testing.T, globalDir string, cfg *GlobalConfig) {
	t.Helper()
	configPath := filepath.Join(globalDir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal global config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}
}

// writeProjectConfig writes a GlobalConfig to the project config file
func writeProjectConfig(t *testing.T, projectDir string, cfg *GlobalConfig) {
	t.Helper()
	configPath := filepath.Join(projectDir, ".addt.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal project config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write project config: %v", err)
	}
}

func TestLoadConfig_DefaultsOnly(t *testing.T) {
	_, _, cleanup := setupTestEnv(t)
	defer cleanup()

	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)

	if cfg.NodeVersion != "20" {
		t.Errorf("NodeVersion = %q, want %q (default)", cfg.NodeVersion, "20")
	}
	if cfg.GoVersion != "1.21" {
		t.Errorf("GoVersion = %q, want %q (default)", cfg.GoVersion, "1.21")
	}
	if cfg.PortRangeStart != 30000 {
		t.Errorf("PortRangeStart = %d, want %d (default)", cfg.PortRangeStart, 30000)
	}
}

func TestLoadConfig_GlobalOverridesDefault(t *testing.T) {
	globalDir, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Write global config
	writeGlobalConfig(t, globalDir, &GlobalConfig{
		NodeVersion: "18",
		GoVersion:   "1.22",
	})

	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)

	if cfg.NodeVersion != "18" {
		t.Errorf("NodeVersion = %q, want %q (from global)", cfg.NodeVersion, "18")
	}
	if cfg.GoVersion != "1.22" {
		t.Errorf("GoVersion = %q, want %q (from global)", cfg.GoVersion, "1.22")
	}
	// UvVersion should still be default since not in global
	if cfg.UvVersion != "0.1.0" {
		t.Errorf("UvVersion = %q, want %q (default)", cfg.UvVersion, "0.1.0")
	}
}

func TestLoadConfig_ProjectOverridesGlobal(t *testing.T) {
	globalDir, projectDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Write global config
	writeGlobalConfig(t, globalDir, &GlobalConfig{
		NodeVersion: "18",
		GoVersion:   "1.22",
		UvVersion:   "0.2.0",
	})

	// Write project config (overrides some values)
	writeProjectConfig(t, projectDir, &GlobalConfig{
		NodeVersion: "22",
		// GoVersion not set - should use global
	})

	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)

	if cfg.NodeVersion != "22" {
		t.Errorf("NodeVersion = %q, want %q (from project)", cfg.NodeVersion, "22")
	}
	if cfg.GoVersion != "1.22" {
		t.Errorf("GoVersion = %q, want %q (from global, not overridden by project)", cfg.GoVersion, "1.22")
	}
	if cfg.UvVersion != "0.2.0" {
		t.Errorf("UvVersion = %q, want %q (from global)", cfg.UvVersion, "0.2.0")
	}
}

func TestLoadConfig_EnvOverridesAll(t *testing.T) {
	globalDir, projectDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Write global config
	writeGlobalConfig(t, globalDir, &GlobalConfig{
		NodeVersion: "18",
		GoVersion:   "1.22",
	})

	// Write project config
	writeProjectConfig(t, projectDir, &GlobalConfig{
		NodeVersion: "22",
		GoVersion:   "1.23",
	})

	// Set env var (highest precedence)
	os.Setenv("ADDT_NODE_VERSION", "24")

	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)

	if cfg.NodeVersion != "24" {
		t.Errorf("NodeVersion = %q, want %q (from env)", cfg.NodeVersion, "24")
	}
	// GoVersion should still come from project (no env override)
	if cfg.GoVersion != "1.23" {
		t.Errorf("GoVersion = %q, want %q (from project)", cfg.GoVersion, "1.23")
	}
}

func TestLoadConfig_BoolPrecedence(t *testing.T) {
	globalDir, projectDir, cleanup := setupTestEnv(t)
	defer cleanup()

	trueVal := true
	falseVal := false

	// Global: persistent=true
	writeGlobalConfig(t, globalDir, &GlobalConfig{
		Persistent: &trueVal,
	})

	// Project: persistent=false (overrides global)
	writeProjectConfig(t, projectDir, &GlobalConfig{
		Persistent: &falseVal,
	})

	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)

	if cfg.Persistent != false {
		t.Errorf("Persistent = %v, want false (from project)", cfg.Persistent)
	}

	// Now test env override
	os.Setenv("ADDT_PERSISTENT", "true")
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)

	if cfg.Persistent != true {
		t.Errorf("Persistent = %v, want true (from env)", cfg.Persistent)
	}
}

func TestLoadConfig_FirewallModePrecedence(t *testing.T) {
	globalDir, projectDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Default is "strict"
	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.FirewallMode != "strict" {
		t.Errorf("FirewallMode = %q, want %q (default)", cfg.FirewallMode, "strict")
	}

	// Global sets permissive
	writeGlobalConfig(t, globalDir, &GlobalConfig{
		FirewallMode: "permissive",
	})
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.FirewallMode != "permissive" {
		t.Errorf("FirewallMode = %q, want %q (from global)", cfg.FirewallMode, "permissive")
	}

	// Project sets off
	writeProjectConfig(t, projectDir, &GlobalConfig{
		FirewallMode: "off",
	})
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.FirewallMode != "off" {
		t.Errorf("FirewallMode = %q, want %q (from project)", cfg.FirewallMode, "off")
	}

	// Env overrides all
	os.Setenv("ADDT_FIREWALL_MODE", "strict")
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.FirewallMode != "strict" {
		t.Errorf("FirewallMode = %q, want %q (from env)", cfg.FirewallMode, "strict")
	}
}

func TestLoadConfig_ExtensionVersionPrecedence(t *testing.T) {
	globalDir, projectDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Save and clear extension env var
	origClaudeVersion := os.Getenv("ADDT_CLAUDE_VERSION")
	os.Unsetenv("ADDT_CLAUDE_VERSION")
	defer func() {
		if origClaudeVersion != "" {
			os.Setenv("ADDT_CLAUDE_VERSION", origClaudeVersion)
		}
	}()

	// Default for claude is "stable" (set in LoadConfig)
	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.ExtensionVersions["claude"] != "stable" {
		t.Errorf("claude version = %q, want %q (default)", cfg.ExtensionVersions["claude"], "stable")
	}

	// Global config sets version
	writeGlobalConfig(t, globalDir, &GlobalConfig{
		Extensions: map[string]*ExtensionSettings{
			"claude": {Version: "1.0.0"},
		},
	})
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.ExtensionVersions["claude"] != "1.0.0" {
		t.Errorf("claude version = %q, want %q (from global)", cfg.ExtensionVersions["claude"], "1.0.0")
	}

	// Project config overrides
	writeProjectConfig(t, projectDir, &GlobalConfig{
		Extensions: map[string]*ExtensionSettings{
			"claude": {Version: "2.0.0"},
		},
	})
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.ExtensionVersions["claude"] != "2.0.0" {
		t.Errorf("claude version = %q, want %q (from project)", cfg.ExtensionVersions["claude"], "2.0.0")
	}

	// Env var overrides all
	os.Setenv("ADDT_CLAUDE_VERSION", "3.0.0")
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.ExtensionVersions["claude"] != "3.0.0" {
		t.Errorf("claude version = %q, want %q (from env)", cfg.ExtensionVersions["claude"], "3.0.0")
	}
}

func TestLoadConfig_ExtensionAutomountPrecedence(t *testing.T) {
	globalDir, projectDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Save and clear extension env var
	origAutomount := os.Getenv("ADDT_CLAUDE_AUTOMOUNT")
	os.Unsetenv("ADDT_CLAUDE_AUTOMOUNT")
	defer func() {
		if origAutomount != "" {
			os.Setenv("ADDT_CLAUDE_AUTOMOUNT", origAutomount)
		}
	}()

	trueVal := true
	falseVal := false

	// Global: automount=true
	writeGlobalConfig(t, globalDir, &GlobalConfig{
		Extensions: map[string]*ExtensionSettings{
			"claude": {Automount: &trueVal},
		},
	})
	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.ExtensionAutomount["claude"] != true {
		t.Errorf("claude automount = %v, want true (from global)", cfg.ExtensionAutomount["claude"])
	}

	// Project: automount=false
	writeProjectConfig(t, projectDir, &GlobalConfig{
		Extensions: map[string]*ExtensionSettings{
			"claude": {Automount: &falseVal},
		},
	})
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.ExtensionAutomount["claude"] != false {
		t.Errorf("claude automount = %v, want false (from project)", cfg.ExtensionAutomount["claude"])
	}

	// Env: automount=true
	os.Setenv("ADDT_CLAUDE_AUTOMOUNT", "true")
	cfg = LoadConfig("20", "1.21", "0.1.0", 30000)
	if cfg.ExtensionAutomount["claude"] != true {
		t.Errorf("claude automount = %v, want true (from env)", cfg.ExtensionAutomount["claude"])
	}
}

func TestLoadConfig_FullPrecedenceChain(t *testing.T) {
	globalDir, projectDir, cleanup := setupTestEnv(t)
	defer cleanup()

	trueVal := true
	portStart := 35000

	// Set up all levels with different values for different keys
	// This tests that each level correctly applies only to the keys it sets

	writeGlobalConfig(t, globalDir, &GlobalConfig{
		NodeVersion:    "18",       // Will be overridden by project
		GoVersion:      "1.22",     // Will be overridden by env
		UvVersion:      "0.2.0",    // Only set here, should persist
		PortRangeStart: &portStart, // Will be overridden by project
		Persistent:     &trueVal,   // Only set here
	})

	projectPort := 36000
	writeProjectConfig(t, projectDir, &GlobalConfig{
		NodeVersion:    "22",         // Overrides global
		PortRangeStart: &projectPort, // Overrides global
		FirewallMode:   "permissive", // Only set here
	})

	os.Setenv("ADDT_GO_VERSION", "1.23")

	cfg := LoadConfig("20", "1.21", "0.1.0", 30000)

	// Check each value comes from the expected source
	if cfg.NodeVersion != "22" {
		t.Errorf("NodeVersion = %q, want %q (from project)", cfg.NodeVersion, "22")
	}
	if cfg.GoVersion != "1.23" {
		t.Errorf("GoVersion = %q, want %q (from env)", cfg.GoVersion, "1.23")
	}
	if cfg.UvVersion != "0.2.0" {
		t.Errorf("UvVersion = %q, want %q (from global)", cfg.UvVersion, "0.2.0")
	}
	if cfg.PortRangeStart != 36000 {
		t.Errorf("PortRangeStart = %d, want %d (from project)", cfg.PortRangeStart, 36000)
	}
	if cfg.Persistent != true {
		t.Errorf("Persistent = %v, want true (from global)", cfg.Persistent)
	}
	if cfg.FirewallMode != "permissive" {
		t.Errorf("FirewallMode = %q, want %q (from project)", cfg.FirewallMode, "permissive")
	}
}
