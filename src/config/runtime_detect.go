package config

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jedi4ever/addt/provider"
)

// defaultAutoselect is the default provider priority order.
var defaultAutoselect = []string{"orbstack", "rancher", "docker", "podman"}

// getAutoselect returns the provider autoselect order from config or default.
func getAutoselect() []string {
	// Check env var first
	if v := os.Getenv("ADDT_PROVIDER_AUTOSELECT"); v != "" {
		return splitTrimmed(v)
	}

	// Check global config
	cfg := loadGlobalConfig()
	if cfg != nil && cfg.Provider != nil && len(cfg.Provider.Autoselect) > 0 {
		return cfg.Provider.Autoselect
	}

	return defaultAutoselect
}

// splitTrimmed splits a comma-separated string and trims whitespace.
func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// DetectContainerRuntime automatically detects which container runtime to use.
// Priority: explicit ADDT_PROVIDER > autoselect order > podman (fallback)
func DetectContainerRuntime() string {
	// If explicitly set, use that
	if p := os.Getenv("ADDT_PROVIDER"); p != "" {
		return p
	}

	// Iterate over autoselect order
	for _, candidate := range getAutoselect() {
		switch candidate {
		case "orbstack":
			if runtime.GOOS == "darwin" && isOrbstackRunning() {
				return "orbstack"
			}
		case "rancher":
			if provider.HasDockerContext("rancher-desktop") {
				return "rancher"
			}
		case "docker":
			if provider.HasDockerContext("desktop-linux") {
				return "docker"
			}
		case "podman":
			if isPodmanAvailable() {
				return "podman"
			}
		case "bwrap":
			if runtime.GOOS == "linux" && isBwrapAvailable() {
				return "bwrap"
			}
		}
	}

	// Default fallback
	return "podman"
}

// EnsureContainerRuntime ensures a container runtime is available.
// Downloads Podman automatically if needed (unless another provider is explicitly selected).
func EnsureContainerRuntime() (string, error) {
	p := os.Getenv("ADDT_PROVIDER")

	// Handle explicitly selected providers
	switch p {
	case "orbstack":
		if !isOrbstackRunning() {
			return "", fmt.Errorf("OrbStack is explicitly selected but not running")
		}
		return "orbstack", nil
	case "docker":
		if !provider.HasDockerContext("desktop-linux") {
			return "", fmt.Errorf("Docker Desktop is explicitly selected but desktop-linux context not found")
		}
		return "docker", nil
	case "rancher":
		if !provider.HasDockerContext("rancher-desktop") {
			return "", fmt.Errorf("Rancher Desktop is explicitly selected but rancher-desktop context not found")
		}
		return "rancher", nil
	case "bwrap":
		if runtime.GOOS != "linux" {
			return "", fmt.Errorf("bwrap provider is Linux-only (current OS: %s)", runtime.GOOS)
		}
		if !isBwrapAvailable() {
			return "", fmt.Errorf("bwrap is explicitly selected but bubblewrap is not installed")
		}
		return "bwrap", nil
	}

	// If explicitly set to something else (e.g. podman), honour it
	if p != "" {
		// Fall through to podman handling below
	}

	// Auto-detect: try autoselect order
	detected := DetectContainerRuntime()
	if detected != "podman" {
		return detected, nil
	}

	// Check if Podman binary exists (even if machine not running)
	podmanPath := GetPodmanPath()
	if podmanPath != "" {
		// Podman binary exists - ensure machine is running on macOS
		if runtime.GOOS == "darwin" {
			if err := ensurePodmanMachine(podmanPath); err != nil {
				return "", fmt.Errorf("failed to start Podman machine: %w", err)
			}
		}
		return "podman", nil
	}

	// Neither available - auto-download Podman
	fmt.Println("No container runtime found. Downloading Podman...")
	if err := DownloadPodman(); err != nil {
		return "", fmt.Errorf("failed to download Podman: %w", err)
	}

	// Verify it works now
	if isPodmanAvailable() {
		return "podman", nil
	}

	return "", fmt.Errorf("Podman downloaded but not working")
}

// isOrbstackRunning checks if OrbStack is installed and running
func isOrbstackRunning() bool {
	orbctlPath, err := exec.LookPath("orbctl")
	if err != nil {
		return false
	}

	cmd := exec.Command(orbctlPath, "status")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "Running"
}

// isPodmanAvailable checks if Podman is available and functional
// Checks both system Podman and bundled Podman
// On macOS, also verifies that a machine is running
func isPodmanAvailable() bool {
	podmanPath := GetPodmanPath()
	if podmanPath == "" {
		return false
	}

	// Check version works
	cmd := exec.Command(podmanPath, "version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if cmd.Run() != nil {
		return false
	}

	// On macOS, we need a machine running
	if runtime.GOOS == "darwin" {
		return isPodmanMachineRunning(podmanPath)
	}

	return true
}

// isPodmanMachineRunning checks if a Podman machine is running (macOS)
func isPodmanMachineRunning(podmanPath string) bool {
	cmd := exec.Command(podmanPath, "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "true")
}

// EnsurePodmanMachineRunning ensures the Podman machine is running (macOS only)
// This should be called before any Podman commands that require the machine
func EnsurePodmanMachineRunning() error {
	if runtime.GOOS != "darwin" {
		return nil // Only needed on macOS
	}

	podmanPath := GetPodmanPath()
	if podmanPath == "" {
		return fmt.Errorf("podman not found")
	}

	return ensurePodmanMachine(podmanPath)
}

// GetPodmanPath returns the path to Podman binary (system or bundled)
func GetPodmanPath() string {
	// First check system Podman (installed via Homebrew, package manager, etc.)
	if path, err := exec.LookPath("podman"); err == nil {
		return path
	}

	// On macOS, the bundled binary is remote-only and can't run machines
	// So we only use it on Linux
	if runtime.GOOS == "darwin" {
		return ""
	}

	// Check bundled Podman (Linux only)
	bundledPath := GetBundledPodmanPath()
	if bundledPath != "" {
		if _, err := os.Stat(bundledPath); err == nil {
			return bundledPath
		}
	}

	return ""
}

// GetRuntimeInfo returns information about the detected runtime
func GetRuntimeInfo() (rt string, version string, extras []string) {
	rt = DetectContainerRuntime()

	switch rt {
	case "docker", "rancher":
		version = getDockerVersion()
	case "orbstack":
		version = getOrbstackVersion()
	case "podman":
		version = getPodmanVersion()
		if hasPasta() {
			extras = append(extras, "pasta")
		}
	case "bwrap":
		version = getBwrapVersion()
	}

	return rt, version, extras
}

func getOrbstackVersion() string {
	orbctlPath, err := exec.LookPath("orbctl")
	if err != nil {
		return "unknown"
	}
	cmd := exec.Command(orbctlPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func getDockerVersion() string {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func getPodmanVersion() string {
	podmanPath := GetPodmanPath()
	if podmanPath == "" {
		return "unknown"
	}
	// Use --version flag which works without daemon connection
	cmd := exec.Command(podmanPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	// Parse "podman version X.Y.Z" -> "X.Y.Z"
	version := strings.TrimSpace(string(output))
	return strings.TrimPrefix(version, "podman version ")
}

func hasPasta() bool {
	_, err := exec.LookPath("pasta")
	return err == nil
}

// isBwrapAvailable checks if bubblewrap is installed and functional
func isBwrapAvailable() bool {
	if _, err := exec.LookPath("bwrap"); err != nil {
		return false
	}
	cmd := exec.Command("bwrap", "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func getBwrapVersion() string {
	cmd := exec.Command("bwrap", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	// Parse "bubblewrap X.Y.Z" -> "X.Y.Z"
	version := strings.TrimSpace(string(output))
	return strings.TrimPrefix(version, "bubblewrap ")
}
