package orbstack

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedi4ever/addt/provider"
)

// GetStatus returns a status string for display
func (p *OrbStackProvider) GetStatus(cfg *provider.Config, envName string) string {
	var parts []string

	// Provider name
	parts = append(parts, "orbstack")

	// Resource limits
	resources := buildResourceString(cfg)
	if resources != "" {
		parts = append(parts, resources)
	}

	// Get Node version from image labels
	cmd := p.dockerCmd("inspect", cfg.ImageName, "--format", "{{index .Config.Labels \"tools.node.version\"}}")
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
		if cfg.WorkdirReadonly {
			parts = append(parts, fmt.Sprintf("%s [RO]", workdir))
		} else {
			parts = append(parts, fmt.Sprintf("%s [RW]", workdir))
		}
	} else {
		parts = append(parts, "[not mounted]")
	}

	// Only show enabled features (skip disabled ones to reduce noise)
	if os.Getenv("GH_TOKEN") != "" {
		parts = append(parts, "GH")
	}

	if cfg.SSHForwardKeys {
		parts = append(parts, fmt.Sprintf("SSH:%s", cfg.SSHForwardMode))
	}

	if cfg.GPGForward != "" && cfg.GPGForward != "off" && cfg.GPGForward != "false" {
		parts = append(parts, fmt.Sprintf("GPG:%s", cfg.GPGForward))
	}

	switch cfg.DockerDindMode {
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

// buildResourceString builds a compact cpu/mem resource string
func buildResourceString(cfg *provider.Config) string {
	var res []string
	if cfg.ContainerCPUs != "" {
		res = append(res, fmt.Sprintf("cpu:%s", cfg.ContainerCPUs))
	}
	if cfg.ContainerMemory != "" {
		res = append(res, fmt.Sprintf("mem:%s", cfg.ContainerMemory))
	}
	return strings.Join(res, " ")
}
