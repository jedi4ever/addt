package orbstack

import (
	"embed"
	"fmt"
	"os"
	"os/exec"

	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/provider"
)

// OrbStackProvider implements the Provider interface for OrbStack
type OrbStackProvider struct {
	config                 *provider.Config
	tempDirs               []string
	sshProxy               *security.SSHProxyAgent
	gpgProxy               *security.GPGProxyAgent
	tmuxProxy              *tmuxProxy
	embeddedDockerfile     []byte
	embeddedDockerfileBase []byte
	embeddedEntrypoint     []byte
	embeddedInitFirewall   []byte
	embeddedInstallSh      []byte
	embeddedExtensions     embed.FS
}

// NewOrbStackProvider creates a new OrbStack provider
func NewOrbStackProvider(cfg *provider.Config, dockerfile, dockerfileBase, entrypoint, initFirewall, installSh []byte, extensions embed.FS) (provider.Provider, error) {
	return &OrbStackProvider{
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

// Initialize initializes the OrbStack provider
func (p *OrbStackProvider) Initialize(cfg *provider.Config) error {
	p.config = cfg

	// Clean up stale temp directories from previous runs
	security.CleanupAll()

	return p.CheckPrerequisites()
}

// GetName returns the provider name
func (p *OrbStackProvider) GetName() string {
	return "orbstack"
}

// CheckPrerequisites verifies OrbStack and Docker CLI are installed and OrbStack is running
func (p *OrbStackProvider) CheckPrerequisites() error {
	// Check orbctl is installed
	if _, err := exec.LookPath("orbctl"); err != nil {
		return fmt.Errorf("OrbStack is not installed. Please install OrbStack from: https://orbstack.dev/")
	}

	// Check OrbStack is running
	cmd := exec.Command("orbctl", "status")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("OrbStack is not running. Please start OrbStack and try again")
	}
	if status := string(output); len(status) > 0 && status != "Running\n" && status != "Running\r\n" {
		return fmt.Errorf("OrbStack is not running (status: %s). Please start OrbStack and try again", status)
	}

	// Check Docker CLI is available (OrbStack provides Docker compatibility)
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker CLI is not available. OrbStack should provide this - try reinstalling OrbStack")
	}

	return nil
}

// Container lifecycle methods (Exists, IsRunning, Start, Stop, Remove, List)
// and name generation (GenerateContainerName, GenerateEphemeralName, GeneratePersistentName)
// are defined in persistent.go

// dockerCmd creates an exec.Cmd for docker targeting the "orbstack" context.
func (p *OrbStackProvider) dockerCmd(args ...string) *exec.Cmd {
	return provider.DockerCmd("orbstack", args...)
}

// Cleanup removes temporary directories and stops proxies
func (p *OrbStackProvider) Cleanup() error {
	// Stop SSH proxy if running
	if p.sshProxy != nil {
		p.sshProxy.Stop()
		p.sshProxy = nil
	}

	// Stop GPG proxy if running
	if p.gpgProxy != nil {
		p.gpgProxy.Stop()
		p.gpgProxy = nil
	}

	// Stop tmux proxy if running
	if p.tmuxProxy != nil {
		p.tmuxProxy.Stop()
		p.tmuxProxy = nil
	}

	for _, dir := range p.tempDirs {
		os.RemoveAll(dir)
	}
	p.tempDirs = []string{}
	return nil
}
