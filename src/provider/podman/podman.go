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

// CheckPrerequisites verifies Podman is installed, offering to install if not found
func (p *PodmanProvider) CheckPrerequisites() error {
	podmanPath := GetPodmanPath()

	// Check if Podman is installed (either system or addt-managed)
	if _, err := exec.LookPath(podmanPath); err != nil {
		// Also check absolute path for addt-managed podman
		if podmanPath != "podman" {
			if _, err := os.Stat(podmanPath); err != nil {
				return p.handlePodmanNotFound()
			}
		} else {
			return p.handlePodmanNotFound()
		}
	}

	// Podman is daemonless, so we just verify it can run
	cmd := exec.Command(podmanPath, "info", "--format", "{{.Host.Os}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Podman is not working correctly. Please check your Podman installation.\n\n%s", GetInstallInstructions())
	}

	return nil
}

// handlePodmanNotFound prompts the user to install Podman
func (p *PodmanProvider) handlePodmanNotFound() error {
	fmt.Println("Podman is not installed.")
	fmt.Println()
	fmt.Println(GetInstallInstructions())
	fmt.Println()

	if PromptInstallPodman() {
		if err := InstallPodman(); err != nil {
			return fmt.Errorf("failed to install Podman: %w", err)
		}
		fmt.Println("\n✓ Podman installed successfully!")
		return nil
	}

	return fmt.Errorf("Podman is required. Please install it and try again")
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
