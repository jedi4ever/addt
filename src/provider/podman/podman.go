package podman

import (
	"embed"
	"fmt"
	"os"
	"os/exec"

	"github.com/jedi4ever/addt/provider"
)

// PodmanProvider implements the Provider interface for Podman
type PodmanProvider struct {
	config                 *provider.Config
	tempDirs               []string
	embeddedDockerfile     []byte
	embeddedDockerfileBase []byte
	embeddedEntrypoint     []byte
	embeddedInitFirewall   []byte
	embeddedInstallSh      []byte
	embeddedExtensions     embed.FS
}

// NewPodmanProvider creates a new Podman provider
func NewPodmanProvider(cfg *provider.Config, dockerfile, dockerfileBase, entrypoint, initFirewall, installSh []byte, extensions embed.FS) (provider.Provider, error) {
	return &PodmanProvider{
		config:                 cfg,
		tempDirs:               []string{},
		embeddedDockerfile:     dockerfile,
		embeddedDockerfileBase: dockerfileBase,
		embeddedEntrypoint:     entrypoint,
		embeddedInitFirewall:   initFirewall,
		embeddedInstallSh:      installSh,
		embeddedExtensions:     extensions,
	}, nil
}

// Initialize initializes the Podman provider
func (p *PodmanProvider) Initialize(cfg *provider.Config) error {
	p.config = cfg
	return p.CheckPrerequisites()
}

// GetName returns the provider name
func (p *PodmanProvider) GetName() string {
	return "podman"
}

// CheckPrerequisites verifies Podman is installed
func (p *PodmanProvider) CheckPrerequisites() error {
	// Check Podman is installed
	if _, err := exec.LookPath("podman"); err != nil {
		return fmt.Errorf("Podman is not installed. Please install Podman from: https://podman.io/getting-started/installation")
	}

	// Verify Podman works (no daemon needed unlike Docker)
	cmd := exec.Command("podman", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Podman is not working properly: %w", err)
	}

	return nil
}

// CheckPastaAvailable checks if pasta is available for network namespaces
func (p *PodmanProvider) CheckPastaAvailable() bool {
	_, err := exec.LookPath("pasta")
	return err == nil
}

// Container lifecycle methods (Exists, IsRunning, Start, Stop, Remove, List)
// and name generation (GenerateContainerName, GenerateEphemeralName, GeneratePersistentName)
// are defined in persistent.go

// Cleanup removes temporary directories
func (p *PodmanProvider) Cleanup() error {
	for _, dir := range p.tempDirs {
		os.RemoveAll(dir)
	}
	p.tempDirs = []string{}
	return nil
}
