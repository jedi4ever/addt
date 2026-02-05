package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jedi4ever/addt/provider"
)

// GetStatus returns a status string for display
func (p *DockerProvider) GetStatus(cfg *provider.Config, envName string) string {
	var parts []string

	// Get Node version from image labels
	cmd := exec.Command("docker", "inspect", cfg.ImageName, "--format", "{{index .Config.Labels \"tools.node.version\"}}")
	if output, err := cmd.Output(); err == nil {
		if nodeVersion := strings.TrimSpace(string(output)); nodeVersion != "" {
			parts = append(parts, fmt.Sprintf("Node %s", nodeVersion))
		}
	}

	// Show mounted workdir with RW/RO/none indicator (key security boundary)
	workdir := cfg.Workdir
	if workdir == "" {
		workdir, _ = os.Getwd()
	}
	if cfg.WorkdirAutomount {
		parts = append(parts, fmt.Sprintf("%s [RW]", workdir))
	} else {
		parts = append(parts, "[not mounted]")
	}

	// Only show enabled features (skip disabled ones to reduce noise)
	if os.Getenv("GH_TOKEN") != "" {
		parts = append(parts, "GH")
	}

	switch cfg.SSHForward {
	case "agent":
		parts = append(parts, "SSH:agent")
	case "keys":
		parts = append(parts, "SSH:keys")
	}

	if cfg.GPGForward != "" && cfg.GPGForward != "off" && cfg.GPGForward != "false" {
		parts = append(parts, fmt.Sprintf("GPG:%s", cfg.GPGForward))
	}

	switch cfg.DindMode {
	case "isolated", "true":
		parts = append(parts, "DinD:isolated")
	case "host":
		parts = append(parts, "DinD:host")
	}

	if cfg.FirewallEnabled {
		parts = append(parts, fmt.Sprintf("Firewall:%s", cfg.FirewallMode))
	}

	if cfg.Persistent {
		parts = append(parts, "Persistent")
	}

	return strings.Join(parts, " | ")
}
