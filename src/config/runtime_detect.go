package config

import (
	"os"
	"os/exec"
	"strings"
)

// DetectContainerRuntime automatically detects which container runtime to use
// Priority: explicit ADDT_PROVIDER > Docker (if running) > Podman (if available) > Docker (default)
func DetectContainerRuntime() string {
	// If explicitly set, use that
	if provider := os.Getenv("ADDT_PROVIDER"); provider != "" {
		return provider
	}

	// Check if Docker is available and running
	if isDockerRunning() {
		return "docker"
	}

	// Check if Podman is available
	if isPodmanAvailable() {
		return "podman"
	}

	// Default to docker (will fail with helpful error if not available)
	return "docker"
}

// isDockerRunning checks if Docker daemon is running
func isDockerRunning() bool {
	// First check if docker command exists
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return false
	}

	// Check if daemon is responsive
	cmd := exec.Command(dockerPath, "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// isPodmanAvailable checks if Podman is available (no daemon needed)
// Checks both system Podman and bundled Podman
func isPodmanAvailable() bool {
	podmanPath := GetPodmanPath()
	if podmanPath == "" {
		return false
	}

	// Podman doesn't need a daemon, just check version works
	cmd := exec.Command(podmanPath, "version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// GetPodmanPath returns the path to Podman binary (system or bundled)
func GetPodmanPath() string {
	// First check system Podman
	if path, err := exec.LookPath("podman"); err == nil {
		return path
	}

	// Check bundled Podman
	bundledPath := GetBundledPodmanPath()
	if bundledPath != "" {
		if _, err := os.Stat(bundledPath); err == nil {
			return bundledPath
		}
	}

	return ""
}

// GetRuntimeInfo returns information about the detected runtime
func GetRuntimeInfo() (runtime string, version string, extras []string) {
	runtime = DetectContainerRuntime()

	switch runtime {
	case "docker":
		version = getDockerVersion()
	case "podman":
		version = getPodmanVersion()
		if hasPasta() {
			extras = append(extras, "pasta")
		}
	}

	return runtime, version, extras
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
