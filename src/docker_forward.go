package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// HandleDockerForwarding configures Docker-in-Docker or host Docker socket forwarding
func HandleDockerForwarding(cfg *Config, containerName string) []string {
	var args []string

	if cfg.DockerForward == "host" {
		socketPath := "/var/run/docker.sock"
		if _, err := os.Stat(socketPath); err == nil {
			args = append(args, "-v", fmt.Sprintf("%s:%s", socketPath, socketPath))

			// Get socket group ID using stat command for cross-platform compatibility
			gid := getDockerSocketGID(socketPath)
			if gid > 0 {
				args = append(args, "--group-add", fmt.Sprintf("%d", gid))
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
		} else {
			fmt.Println("Warning: DCLAUDE_DOCKER_FORWARD=host but /var/run/docker.sock not found")
		}
	} else if cfg.DockerForward == "isolated" || cfg.DockerForward == "true" {
		args = append(args, "--privileged")
		args = append(args, "-v", fmt.Sprintf("dclaude-docker-%s:/var/lib/docker", containerName))
		args = append(args, "-e", "DCLAUDE_DIND=true")
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
