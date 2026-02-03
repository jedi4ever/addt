package cmd

import (
	"testing"

	"github.com/jedi4ever/addt/provider"
)

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	buildCalled     bool
	rebuildArg      bool
	rebuildBaseArg  bool
	imageNameCalled bool
}

func (m *mockProvider) Initialize(cfg *provider.Config) error      { return nil }
func (m *mockProvider) Run(spec *provider.RunSpec) error           { return nil }
func (m *mockProvider) Shell(spec *provider.RunSpec) error         { return nil }
func (m *mockProvider) Cleanup() error                             { return nil }
func (m *mockProvider) Exists(name string) bool                    { return false }
func (m *mockProvider) IsRunning(name string) bool                 { return false }
func (m *mockProvider) Start(name string) error                    { return nil }
func (m *mockProvider) Stop(name string) error                     { return nil }
func (m *mockProvider) Remove(name string) error                   { return nil }
func (m *mockProvider) List() ([]provider.Environment, error)      { return nil, nil }
func (m *mockProvider) GeneratePersistentName() string             { return "test-persistent" }
func (m *mockProvider) GenerateEphemeralName() string              { return "test-ephemeral" }
func (m *mockProvider) GetStatus(cfg *provider.Config, name string) string { return "test" }
func (m *mockProvider) GetName() string                            { return "mock" }
func (m *mockProvider) GetExtensionEnvVars(imageName string) []string { return nil }

func (m *mockProvider) DetermineImageName() string {
	m.imageNameCalled = true
	return "test-image"
}

func (m *mockProvider) BuildIfNeeded(rebuild bool, rebuildBase bool) error {
	m.buildCalled = true
	m.rebuildArg = rebuild
	m.rebuildBaseArg = rebuildBase
	return nil
}

func TestHandleBuildCommand_Basic(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{}, false)

	if !mock.buildCalled {
		t.Error("BuildIfNeeded was not called")
	}

	if !mock.rebuildArg {
		t.Error("rebuild should be true for build command")
	}

	if mock.rebuildBaseArg {
		t.Error("rebuildBase should be false")
	}

	if !mock.imageNameCalled {
		t.Error("DetermineImageName was not called")
	}
}

func TestHandleBuildCommand_NoCache(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{}, true)

	if !cfg.NoCache {
		t.Error("NoCache should be true when noCache=true")
	}
}

func TestHandleBuildCommand_BuildArgExtensions(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{"--build-arg", "ADDT_EXTENSIONS=claude,codex"}, false)

	if cfg.Extensions != "claude,codex" {
		t.Errorf("Extensions = %q, want %q", cfg.Extensions, "claude,codex")
	}
}

func TestHandleBuildCommand_BuildArgNodeVersion(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{"--build-arg", "NODE_VERSION=20"}, false)

	if cfg.NodeVersion != "20" {
		t.Errorf("NodeVersion = %q, want %q", cfg.NodeVersion, "20")
	}
}

func TestHandleBuildCommand_BuildArgGoVersion(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{"--build-arg", "GO_VERSION=1.22"}, false)

	if cfg.GoVersion != "1.22" {
		t.Errorf("GoVersion = %q, want %q", cfg.GoVersion, "1.22")
	}
}

func TestHandleBuildCommand_BuildArgUvVersion(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{"--build-arg", "UV_VERSION=0.2.0"}, false)

	if cfg.UvVersion != "0.2.0" {
		t.Errorf("UvVersion = %q, want %q", cfg.UvVersion, "0.2.0")
	}
}

func TestHandleBuildCommand_BuildArgExtensionVersion(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{"--build-arg", "CLAUDE_VERSION=1.0.5"}, false)

	if cfg.ExtensionVersions == nil {
		t.Fatal("ExtensionVersions is nil")
	}

	if cfg.ExtensionVersions["claude"] != "1.0.5" {
		t.Errorf("ExtensionVersions[claude] = %q, want %q", cfg.ExtensionVersions["claude"], "1.0.5")
	}
}

func TestHandleBuildCommand_MultipleBuildArgs(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	args := []string{
		"--build-arg", "NODE_VERSION=18",
		"--build-arg", "GO_VERSION=1.21",
		"--build-arg", "ADDT_EXTENSIONS=claude",
		"--build-arg", "CLAUDE_VERSION=2.0.0",
	}

	HandleBuildCommand(mock, cfg, args, false)

	if cfg.NodeVersion != "18" {
		t.Errorf("NodeVersion = %q, want %q", cfg.NodeVersion, "18")
	}

	if cfg.GoVersion != "1.21" {
		t.Errorf("GoVersion = %q, want %q", cfg.GoVersion, "1.21")
	}

	if cfg.Extensions != "claude" {
		t.Errorf("Extensions = %q, want %q", cfg.Extensions, "claude")
	}

	if cfg.ExtensionVersions["claude"] != "2.0.0" {
		t.Errorf("ExtensionVersions[claude] = %q, want %q", cfg.ExtensionVersions["claude"], "2.0.0")
	}
}

func TestHandleBuildCommand_SetsImageName(t *testing.T) {
	mock := &mockProvider{}
	cfg := &provider.Config{}

	HandleBuildCommand(mock, cfg, []string{}, false)

	if cfg.ImageName != "test-image" {
		t.Errorf("ImageName = %q, want %q", cfg.ImageName, "test-image")
	}
}
