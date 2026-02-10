package bwrap

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedi4ever/addt/provider"
)

// GetStatus returns a status string for display
func (b *BwrapProvider) GetStatus(cfg *provider.Config, envName string) string {
	var parts []string

	// Provider name
	parts = append(parts, "bwrap (lightweight)")

	// Resource limits
	resources := buildBwrapResourceString(cfg)
	if resources != "" {
		parts = append(parts, resources)
	}

	// Show mounted workdir with RW/RO/none indicator
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

	// Only show enabled features
	if os.Getenv("GH_TOKEN") != "" {
		parts = append(parts, "GH")
	}

	if cfg.SSHForwardKeys {
		parts = append(parts, fmt.Sprintf("SSH:%s", cfg.SSHForwardMode))
	}

	if cfg.GPGForward != "" && cfg.GPGForward != "off" && cfg.GPGForward != "false" {
		parts = append(parts, fmt.Sprintf("GPG:%s", cfg.GPGForward))
	}

	// Unsupported features noted
	if cfg.DockerDindMode != "" && cfg.DockerDindMode != "off" {
		parts = append(parts, "DinD:unsupported")
	}

	if cfg.FirewallEnabled {
		parts = append(parts, "Firewall:unsupported")
	}

	sec := cfg.Security
	if sec.NetworkMode == "none" {
		parts = append(parts, "Net:isolated")
	}

	return strings.Join(parts, " | ")
}

// buildBwrapResourceString builds a compact cpu/mem resource string
func buildBwrapResourceString(cfg *provider.Config) string {
	var res []string
	if cfg.ContainerCPUs != "" {
		res = append(res, fmt.Sprintf("cpu:%s", cfg.ContainerCPUs))
	}
	if cfg.ContainerMemory != "" {
		res = append(res, fmt.Sprintf("mem:%s", cfg.ContainerMemory))
	}
	return strings.Join(res, " ")
}
