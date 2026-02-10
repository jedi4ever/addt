package bwrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
)

// BwrapProvider implements the Provider interface using bubblewrap (bwrap).
// This is a lightweight Linux-only sandbox that uses kernel namespaces directly,
// without requiring a container runtime like Docker or Podman.
//
// Supported features:
//   - Filesystem isolation (mount namespaces)
//   - PID namespace isolation
//   - IPC namespace isolation
//   - Network isolation (shared or fully isolated)
//   - Working directory mounting (RO/RW)
//   - Environment variable control
//   - SSH/GPG/Tmux forwarding (direct socket access)
//   - Shell history persistence
//   - Process limits (ulimits via wrapper)
//   - Read-only root filesystem
//
// NOT supported (inherent bwrap limitations):
//   - Image building (uses host-installed tools directly)
//   - Docker-in-Docker / nested containers
//   - Port mapping (ports are directly accessible on host network)
//   - Persistent containers (no stop/restart; ephemeral processes only)
//   - Firewall rules (only full network isolation via --unshare-net)
//   - Extension installation (extensions must be pre-installed on host)
//   - Seccomp profiles (bwrap uses raw BPF, not Docker JSON format)
//   - Secret isolation (no two-step copy pattern; secrets passed as env vars)
type BwrapProvider struct {
	config   *provider.Config
	tempDirs []string
	sshProxy *security.SSHProxyAgent
	gpgProxy *security.GPGProxyAgent
}

// NewBwrapProvider creates a new bubblewrap provider
func NewBwrapProvider(cfg *provider.Config) (provider.Provider, error) {
	return &BwrapProvider{
		config:   cfg,
		tempDirs: []string{},
	}, nil
}

// Initialize initializes the bwrap provider
func (b *BwrapProvider) Initialize(cfg *provider.Config) error {
	b.config = cfg
	security.CleanupAll()
	return b.CheckPrerequisites()
}

// GetName returns the provider name
func (b *BwrapProvider) GetName() string {
	return "bwrap"
}

// CheckPrerequisites verifies bubblewrap is installed and we're on Linux
func (b *BwrapProvider) CheckPrerequisites() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("bubblewrap provider is Linux-only (current OS: %s)", runtime.GOOS)
	}

	if _, err := exec.LookPath("bwrap"); err != nil {
		return fmt.Errorf("bubblewrap (bwrap) is not installed.\n" +
			"Install it with:\n" +
			"  Ubuntu/Debian: sudo apt install bubblewrap\n" +
			"  Fedora:        sudo dnf install bubblewrap\n" +
			"  Arch:          sudo pacman -S bubblewrap")
	}

	// Verify bwrap works
	cmd := exec.Command("bwrap", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bubblewrap is not working properly: %w", err)
	}

	return nil
}

// Cleanup removes temporary directories and stops proxies
func (b *BwrapProvider) Cleanup() error {
	if b.sshProxy != nil {
		b.sshProxy.Stop()
		b.sshProxy = nil
	}
	if b.gpgProxy != nil {
		b.gpgProxy.Stop()
		b.gpgProxy = nil
	}
	for _, dir := range b.tempDirs {
		os.RemoveAll(dir)
	}
	b.tempDirs = []string{}
	return nil
}

// getHomeDir returns the persistent home directory for bwrap sandboxes
func (b *BwrapProvider) getHomeDir() string {
	addtHome := util.GetAddtHome()
	if addtHome == "" {
		return filepath.Join(os.TempDir(), "addt-bwrap-home")
	}
	return filepath.Join(addtHome, "bwrap", "home")
}

// ensureHomeDir creates the persistent home directory if it doesn't exist
func (b *BwrapProvider) ensureHomeDir() error {
	return os.MkdirAll(b.getHomeDir(), 0700)
}
