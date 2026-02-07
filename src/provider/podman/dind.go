package podman

import (
	"fmt"
	"os"
	"path/filepath"
)

// HandlePodmanForwarding configures Podman-in-Podman (nested containers) support.
// Modes:
//   - "host": Share host's Podman socket (dangerous but useful for some workflows)
//   - "isolated" or "true": Run isolated nested Podman inside container (requires --privileged)
//   - "" or other: No Podman forwarding
func (p *PodmanProvider) HandlePodmanForwarding(mode string, containerName string) []string {
	switch mode {
	case "isolated", "true":
		return p.handleIsolatedPodmanForwarding(containerName)
	case "host":
		return p.handleHostPodmanForwarding()
	default:
		return nil
	}
}

// handleIsolatedPodmanForwarding configures an isolated nested Podman inside the container
func (p *PodmanProvider) handleIsolatedPodmanForwarding(containerName string) []string {
	var args []string

	// Isolated mode requires privileged access for nested user namespaces
	args = append(args, "--privileged")

	// Use a named volume for Podman container storage persistence
	volumeName := fmt.Sprintf("addt-podman-%s", containerName)
	args = append(args, "-v", fmt.Sprintf("%s:/home/addt/.local/share/containers", volumeName))

	// Signal to entrypoint that it should set up nested Podman
	args = append(args, "-e", "ADDT_DOCKER_DIND_ENABLE=true")

	return args
}

// handleHostPodmanForwarding shares the host's Podman socket with the container
func (p *PodmanProvider) handleHostPodmanForwarding() []string {
	var args []string

	podmanSocket := os.Getenv("XDG_RUNTIME_DIR")
	if podmanSocket == "" {
		podmanSocket = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	socketPath := filepath.Join(podmanSocket, "podman", "podman.sock")

	if _, err := os.Stat(socketPath); err == nil {
		args = append(args,
			"-v", fmt.Sprintf("%s:/run/podman/podman.sock", socketPath),
			"-e", "DOCKER_HOST=unix:///run/podman/podman.sock",
		)
	}

	return args
}
