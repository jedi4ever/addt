package podman

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// HandleDockerForwarding configures Docker-in-Docker or host Docker socket forwarding.
// Modes:
//   - "host": Mount host's Docker/Podman socket (shares runtime with host)
//   - "isolated" or "true": Run isolated Docker daemon inside container (requires --privileged)
//   - "" or other: No Docker forwarding
func (p *PodmanProvider) HandleDockerForwarding(dindMode, containerName string) []string {
	switch dindMode {
	case "host":
		return p.handleHostDockerForwarding()
	case "isolated", "true":
		return p.handleIsolatedDockerForwarding(containerName)
	default:
		return nil
	}
}

// handleHostDockerForwarding mounts the host's Docker/Podman socket into the container
func (p *PodmanProvider) handleHostDockerForwarding() []string {
	var args []string

	// Try Docker socket first, then Podman socket
	socketPath := "/var/run/docker.sock"
	if _, err := os.Stat(socketPath); err != nil {
		// Try Podman socket locations
		podmanSockets := []string{
			fmt.Sprintf("/run/user/%d/podman/podman.sock", os.Getuid()),
			"/var/run/podman/podman.sock",
		}
		found := false
		for _, ps := range podmanSockets {
			if _, err := os.Stat(ps); err == nil {
				socketPath = ps
				found = true
				break
			}
		}
		if !found {
			fmt.Println("Warning: ADDT_DIND_MODE=host but no Docker/Podman socket found")
			return args
		}
	}

	// Mount the socket
	args = append(args, "-v", fmt.Sprintf("%s:/var/run/docker.sock", socketPath))

	// Add user to socket's group
	args = append(args, getDockerGroupArgs(socketPath)...)

	return args
}

// handleIsolatedDockerForwarding configures an isolated Docker daemon inside the container
func (p *PodmanProvider) handleIsolatedDockerForwarding(containerName string) []string {
	var args []string

	// Isolated mode requires privileged access
	args = append(args, "--privileged")

	// Use a named volume for Docker data persistence
	volumeName := fmt.Sprintf("addt-docker-%s", containerName)
	args = append(args, "-v", fmt.Sprintf("%s:/var/lib/docker", volumeName))

	// Signal to entrypoint that it should start dockerd
	args = append(args, "-e", "ADDT_DIND=true")

	return args
}

// getDockerGroupArgs returns --group-add arguments for Docker socket access
func getDockerGroupArgs(socketPath string) []string {
	var args []string

	gid := getDockerSocketGID(socketPath)
	if gid > 0 {
		args = append(args, "--group-add", fmt.Sprintf("%d", gid))
		// Add common Docker group IDs as fallbacks
		if gid != 102 {
			args = append(args, "--group-add", "102")
		}
		if gid != 999 {
			args = append(args, "--group-add", "999")
		}
	} else {
		fmt.Println("Warning: Could not detect Docker socket group, using common defaults")
		args = append(args, "--group-add", "102", "--group-add", "999")
	}

	return args
}

// getDockerSocketGID returns the group ID of the Docker socket
func getDockerSocketGID(socketPath string) int {
	// Try using syscall.Stat_t first (works on Linux)
	if info, err := os.Stat(socketPath); err == nil {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			return int(stat.Gid)
		}
	}

	// Fallback: use stat command (works on macOS and Linux)
	// Try GNU stat format first (Linux)
	cmd := exec.Command("stat", "-c", "%g", socketPath)
	if output, err := cmd.Output(); err == nil {
		if gid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			return gid
		}
	}

	// Try BSD stat format (macOS)
	cmd = exec.Command("stat", "-f", "%g", socketPath)
	if output, err := cmd.Output(); err == nil {
		if gid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			return gid
		}
	}

	return 0
}
