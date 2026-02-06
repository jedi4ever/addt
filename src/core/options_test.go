package core

import (
	"testing"

	"github.com/jedi4ever/addt/provider"
)

// mockOptionsProvider for options tests
type mockOptionsProvider struct{}

func (m *mockOptionsProvider) Initialize(cfg *provider.Config) error              { return nil }
func (m *mockOptionsProvider) Run(spec *provider.RunSpec) error                   { return nil }
func (m *mockOptionsProvider) Shell(spec *provider.RunSpec) error                 { return nil }
func (m *mockOptionsProvider) Cleanup() error                                     { return nil }
func (m *mockOptionsProvider) Exists(name string) bool                            { return false }
func (m *mockOptionsProvider) IsRunning(name string) bool                         { return false }
func (m *mockOptionsProvider) Start(name string) error                            { return nil }
func (m *mockOptionsProvider) Stop(name string) error                             { return nil }
func (m *mockOptionsProvider) Remove(name string) error                           { return nil }
func (m *mockOptionsProvider) List() ([]provider.Environment, error)              { return nil, nil }
func (m *mockOptionsProvider) GeneratePersistentName() string                     { return "test-persistent" }
func (m *mockOptionsProvider) GenerateEphemeralName() string                      { return "test-ephemeral" }
func (m *mockOptionsProvider) GetStatus(cfg *provider.Config, name string) string { return "test" }
func (m *mockOptionsProvider) GetName() string                                    { return "mock" }
func (m *mockOptionsProvider) GetExtensionEnvVars(imageName string) []string      { return nil }
func (m *mockOptionsProvider) DetermineImageName() string                         { return "test-image" }
func (m *mockOptionsProvider) BuildIfNeeded(rebuild bool, rebuildBase bool) error { return nil }

func TestBuildRunOptions_Basic(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{"--help"}, false)

	if opts.Name != "test-container" {
		t.Errorf("Name = %q, want 'test-container'", opts.Name)
	}

	if opts.ImageName != "test-image" {
		t.Errorf("ImageName = %q, want 'test-image'", opts.ImageName)
	}

	if len(opts.Args) != 1 || opts.Args[0] != "--help" {
		t.Errorf("Args = %v, want ['--help']", opts.Args)
	}
}

func TestBuildRunOptions_ShellMode(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	// Shell mode with no args
	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, true)

	if len(opts.Args) != 0 {
		t.Errorf("Args = %v, want empty for shell mode", opts.Args)
	}

	// Shell mode with args
	opts = BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{"-c", "ls"}, true)

	if len(opts.Args) != 2 {
		t.Errorf("Args = %v, want ['-c', 'ls'] for shell mode with args", opts.Args)
	}
}

func TestBuildRunOptions_Persistent(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		Persistent:       true,
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, false)

	if !opts.Persistent {
		t.Error("Persistent should be true")
	}
}

func TestBuildRunOptions_SSHAndGPG(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		SSHForward:       "keys",
		GPGForward:       "keys",
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, false)

	if opts.SSHForward != "keys" {
		t.Errorf("SSHForward = %q, want 'keys'", opts.SSHForward)
	}

	if opts.GPGForward != "keys" {
		t.Errorf("GPGForward = %q, want 'keys'", opts.GPGForward)
	}
}

func TestBuildRunOptions_DockerDindMode(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		DockerDindMode:   "isolated",
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, false)

	if opts.DockerDindMode != "isolated" {
		t.Errorf("DockerDindMode = %q, want 'isolated'", opts.DockerDindMode)
	}
}

func TestBuildRunOptions_Resources(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		DockerCPUs:       "2",
		DockerMemory:     "4g",
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, false)

	if opts.DockerCPUs != "2" {
		t.Errorf("DockerCPUs = %q, want '2'", opts.DockerCPUs)
	}

	if opts.DockerMemory != "4g" {
		t.Errorf("DockerMemory = %q, want '4g'", opts.DockerMemory)
	}
}

func TestBuildRunOptions_IncludesVolumes(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, false)

	if len(opts.Volumes) == 0 {
		t.Error("Expected volumes to be set")
	}
}

func TestBuildRunOptions_IncludesPorts(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		Ports:            []string{"3000", "8080"},
		PortRangeStart:   30000,
		WorkdirAutomount: true,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, false)

	if len(opts.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(opts.Ports))
	}
}

func TestBuildRunOptions_IncludesEnv(t *testing.T) {
	cfg := &provider.Config{
		ImageName:        "test-image",
		WorkdirAutomount: true,
		PortRangeStart:   30000,
	}

	opts := BuildRunOptions(&mockOptionsProvider{}, cfg, "test-container", []string{}, false)

	if opts.Env == nil {
		t.Error("Expected env to be set")
	}

	// Should have at least COLUMNS and LINES
	if opts.Env["COLUMNS"] == "" {
		t.Error("COLUMNS not set in env")
	}
}
