package config

import (
	"os"
	"path/filepath"
	"testing"

	cfgtypes "github.com/jedi4ever/addt/config"
)

// setupTestEnv creates temporary directories for testing and sets ADDT_CONFIG_DIR
// Returns cleanup function to restore original state
func setupTestEnv(t *testing.T) (globalDir, projectDir string, cleanup func()) {
	t.Helper()

	// Save original env vars
	origConfigDir := os.Getenv("ADDT_CONFIG_DIR")
	origCwd, _ := os.Getwd()

	// Create temp directories
	globalDir = t.TempDir()
	projectDir = t.TempDir()

	// Set ADDT_CONFIG_DIR to isolate from real home directory
	os.Setenv("ADDT_CONFIG_DIR", globalDir)

	// Change to project directory
	os.Chdir(projectDir)

	cleanup = func() {
		os.Setenv("ADDT_CONFIG_DIR", origConfigDir)
		os.Chdir(origCwd)
	}

	return globalDir, projectDir, cleanup
}

func TestGetGlobalConfigPath(t *testing.T) {
	globalDir, _, cleanup := setupTestEnv(t)
	defer cleanup()

	path := cfgtypes.GetGlobalConfigPath()

	// Check that path ends with config.yaml and contains our temp dir name
	// (avoid macOS /var vs /private/var symlink issues)
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("GetGlobalConfigPath() should end with config.yaml, got %q", path)
	}
	if filepath.Base(filepath.Dir(path)) != filepath.Base(globalDir) {
		t.Errorf("GetGlobalConfigPath() dir = %q, want dir containing %q", filepath.Dir(path), filepath.Base(globalDir))
	}
}

func TestLoadGlobalConfigFile_NonExistent(t *testing.T) {
	_, _, cleanup := setupTestEnv(t)
	defer cleanup()

	cfg, err := cfgtypes.LoadGlobalConfigFile()
	if err != nil {
		t.Fatalf("LoadGlobalConfigFile() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadGlobalConfigFile() returned nil")
	}

	// Should return empty config when file doesn't exist
	if cfg.NodeVersion != "" {
		t.Errorf("Expected empty NodeVersion, got %q", cfg.NodeVersion)
	}
}

func TestSaveAndLoadGlobalConfig(t *testing.T) {
	_, _, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create and save config
	cfg := &cfgtypes.GlobalConfig{
		NodeVersion: "20",
		GoVersion:   "1.21",
		Container: &cfgtypes.ContainerSettings{
			CPUs:   "2",
			Memory: "4g",
		},
	}

	err := cfgtypes.SaveGlobalConfigFile(cfg)
	if err != nil {
		t.Fatalf("SaveGlobalConfigFile() error = %v", err)
	}

	// Load and verify
	loaded, err := cfgtypes.LoadGlobalConfigFile()
	if err != nil {
		t.Fatalf("LoadGlobalConfigFile() error = %v", err)
	}

	if loaded.NodeVersion != "20" {
		t.Errorf("NodeVersion = %q, want %q", loaded.NodeVersion, "20")
	}
	if loaded.GoVersion != "1.21" {
		t.Errorf("GoVersion = %q, want %q", loaded.GoVersion, "1.21")
	}
	if loaded.Container == nil || loaded.Container.CPUs != "2" {
		t.Errorf("Container.CPUs = %q, want %q", loaded.Container.CPUs, "2")
	}
	if loaded.Container == nil || loaded.Container.Memory != "4g" {
		t.Errorf("Container.Memory = %q, want %q", loaded.Container.Memory, "4g")
	}
}
