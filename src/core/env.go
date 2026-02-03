package core

import (
	"fmt"
	"os"

	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util/terminal"
)

// BuildEnvironment creates the environment variables map for the container
func BuildEnvironment(p provider.Provider, cfg *provider.Config) map[string]string {
	env := make(map[string]string)

	// Add extension-required environment variables
	addExtensionEnvVars(env, p, cfg)

	// Add user-configured environment variables
	addUserEnvVars(env, cfg)

	// Add terminal environment variables
	addTerminalEnvVars(env)

	// Add AI context (port mappings for system prompt)
	AddAIContext(env, cfg)

	// Add firewall configuration
	addFirewallEnvVars(env, cfg)

	// Add command override
	addCommandEnvVar(env, cfg)

	return env
}

// addExtensionEnvVars adds environment variables required by extensions
func addExtensionEnvVars(env map[string]string, p provider.Provider, cfg *provider.Config) {
	extensionEnvVars := p.GetExtensionEnvVars(cfg.ImageName)
	for _, varName := range extensionEnvVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}
}

// addUserEnvVars adds user-configured environment variables
func addUserEnvVars(env map[string]string, cfg *provider.Config) {
	for _, varName := range cfg.EnvVars {
		if value := os.Getenv(varName); value != "" {
			env[varName] = value
		}
	}
}

// addTerminalEnvVars adds terminal-related environment variables
func addTerminalEnvVars(env map[string]string) {
	// Pass terminal type for proper rendering
	if term := os.Getenv("TERM"); term != "" {
		env["TERM"] = term
	}
	if colorterm := os.Getenv("COLORTERM"); colorterm != "" {
		env["COLORTERM"] = colorterm
	}

	// Pass terminal size (critical for proper line handling in containers)
	cols, lines := terminal.GetTerminalSize()
	env["COLUMNS"] = fmt.Sprintf("%d", cols)
	env["LINES"] = fmt.Sprintf("%d", lines)
}

// addFirewallEnvVars adds firewall configuration environment variables
func addFirewallEnvVars(env map[string]string, cfg *provider.Config) {
	if cfg.FirewallEnabled {
		env["ADDT_FIREWALL_ENABLED"] = "true"
		env["ADDT_FIREWALL_MODE"] = cfg.FirewallMode
	}
}

// addCommandEnvVar adds the command override environment variable
func addCommandEnvVar(env map[string]string, cfg *provider.Config) {
	if cfg.Command != "" {
		env["ADDT_COMMAND"] = cfg.Command
	}
}
