package provider

import (
	"os"
	"os/exec"
)

// DockerCmd creates an exec.Cmd for docker targeting a specific context.
// This ensures each provider (docker, orbstack) hits the correct daemon
// regardless of which Docker context is currently active.
func DockerCmd(context string, args ...string) *exec.Cmd {
	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ(), "DOCKER_CONTEXT="+context)
	return cmd
}
