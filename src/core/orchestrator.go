package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/dclaude/internal/ports"
	"github.com/jedi4ever/dclaude/internal/terminal"
	"github.com/jedi4ever/dclaude/internal/util"
	"github.com/jedi4ever/dclaude/provider"
)

// Orchestrator coordinates provider operations with business logic
type Orchestrator struct {
	provider provider.Provider
	config   *provider.Config
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(p provider.Provider, cfg *provider.Config) *Orchestrator {
	return &Orchestrator{
		provider: p,
		config:   cfg,
	}
}

// RunClaude orchestrates the Claude Code execution
func (o *Orchestrator) RunClaude(args []string, openShell bool) error {
	// Determine environment name
	var envName string
	if o.config.Persistent {
		envName = o.generatePersistentName()
	} else {
		envName = o.generateEphemeralName()
	}

	// Build RunSpec from config
	spec := o.buildRunSpec(envName, args, openShell)

	// Display status
	o.displayStatus(envName)

	// Execute via provider
	if openShell {
		return o.provider.Shell(spec)
	}
	return o.provider.Run(spec)
}

// buildRunSpec builds a RunSpec from the current configuration
func (o *Orchestrator) buildRunSpec(name string, args []string, openShell bool) *provider.RunSpec {
	cwd, _ := os.Getwd()

	spec := &provider.RunSpec{
		Name:        name,
		ImageName:   o.config.ImageName,
		Args:        args,
		WorkDir:     cwd,
		Interactive: terminal.IsTerminal(),
		Persistent:  o.config.Persistent,
		Volumes:     o.buildVolumes(cwd),
		Ports:       o.buildPorts(),
		Env:         o.buildEnvironment(),
		SSHForward:  o.config.SSHForward,
		GPGForward:  o.config.GPGForward,
		DindMode:    o.config.DindMode,
	}

	// Special handling for shell mode with args
	if openShell && len(args) > 0 {
		spec.Args = args
	} else if openShell {
		spec.Args = []string{}
	} else {
		// Normal mode - pass args directly (entrypoint calls claude)
		spec.Args = args
	}

	// Log command if enabled
	if o.config.LogEnabled {
		util.LogCommand(o.config.LogFile, cwd, name, args)
	}

	// Store env file path if exists
	envFilePath := o.config.EnvFile
	if envFilePath == "" {
		envFilePath = ".env"
	}
	if !filepath.IsAbs(envFilePath) {
		envFilePath = filepath.Join(cwd, envFilePath)
	}
	if info, err := os.Stat(envFilePath); err == nil && !info.IsDir() {
		spec.Env["DCLAUDE_ENV_FILE"] = envFilePath
	}

	return spec
}

// buildVolumes builds volume mounts
func (o *Orchestrator) buildVolumes(cwd string) []provider.VolumeMount {
	var volumes []provider.VolumeMount

	// Only mount working directory if enabled (default: true)
	if o.config.MountWorkdir {
		volumes = append(volumes, provider.VolumeMount{
			Source:   cwd,
			Target:   "/workspace",
			ReadOnly: false,
		})
	}

	return volumes
}

// buildPorts builds port mappings
func (o *Orchestrator) buildPorts() []provider.PortMapping {
	var portsList []provider.PortMapping
	hostPort := o.config.PortRangeStart

	for _, containerPort := range o.config.Ports {
		containerPort = strings.TrimSpace(containerPort)
		hostPort = ports.FindAvailablePort(hostPort)

		// Parse container port as int
		var containerPortInt int
		fmt.Sscanf(containerPort, "%d", &containerPortInt)

		portsList = append(portsList, provider.PortMapping{
			Container: containerPortInt,
			Host:      hostPort,
		})
		hostPort++
	}

	return portsList
}

// buildEnvironment builds environment variables map
func (o *Orchestrator) buildEnvironment() map[string]string {
	env := make(map[string]string)

	// Get extension-required environment variables from the image
	extensionEnvVars := o.provider.GetExtensionEnvVars(o.config.ImageName)
	for _, varName := range extensionEnvVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}

	// Pass user-configured environment variables (can override extension vars)
	for _, varName := range o.config.EnvVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}

	// Pass terminal environment variables for proper paste handling
	if term := os.Getenv("TERM"); term != "" {
		env["TERM"] = term
	}
	if colorterm := os.Getenv("COLORTERM"); colorterm != "" {
		env["COLORTERM"] = colorterm
	}

	// Pass terminal size variables (critical for proper line handling in containers)
	cols, lines := terminal.GetTerminalSize()
	env["COLUMNS"] = fmt.Sprintf("%d", cols)
	env["LINES"] = fmt.Sprintf("%d", lines)

	// Build port map string for display
	if len(o.config.Ports) > 0 {
		var portMappings []string
		hostPort := o.config.PortRangeStart
		for _, containerPort := range o.config.Ports {
			containerPort = strings.TrimSpace(containerPort)
			hostPort = ports.FindAvailablePort(hostPort)
			portMappings = append(portMappings, fmt.Sprintf("%s:%d", containerPort, hostPort))
			hostPort++
		}
		portMapString := strings.Join(portMappings, ",")
		env["DCLAUDE_PORT_MAP"] = portMapString
	}

	// Add firewall configuration
	if o.config.FirewallEnabled {
		env["DCLAUDE_FIREWALL_ENABLED"] = "true"
		env["DCLAUDE_FIREWALL_MODE"] = o.config.FirewallMode
	}

	// Add command override if specified
	if o.config.Command != "" {
		env["DCLAUDE_COMMAND"] = o.config.Command
	}

	return env
}

// displayStatus displays the status line
func (o *Orchestrator) displayStatus(envName string) {
	status := o.provider.GetStatus(o.config, envName)

	// Add port mappings to status
	if len(o.config.Ports) > 0 {
		hostPort := o.config.PortRangeStart
		var portMappings []string
		for _, containerPort := range o.config.Ports {
			containerPort = strings.TrimSpace(containerPort)
			hostPort = ports.FindAvailablePort(hostPort)
			portMappings = append(portMappings, fmt.Sprintf("%s→%d", containerPort, hostPort))
			hostPort++
		}
		portMapDisplay := strings.Join(portMappings, ",")
		status += fmt.Sprintf(" | Ports:%s", portMapDisplay)
	}

	fmt.Printf("✓ %s\n", status)
}

// generatePersistentName generates a persistent container name
func (o *Orchestrator) generatePersistentName() string {
	return o.provider.GeneratePersistentName()
}

// generateEphemeralName generates an ephemeral container name
func (o *Orchestrator) generateEphemeralName() string {
	return o.provider.GenerateEphemeralName()
}
