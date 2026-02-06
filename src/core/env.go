package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/config/otel"
	"github.com/jedi4ever/addt/extensions"
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

	// Add OpenTelemetry configuration
	addOtelEnvVars(env, cfg)

	return env
}

// addExtensionEnvVars adds environment variables required by extensions
// Supports both "VAR_NAME" (pass-through from host) and "VAR_NAME=default" (with default value)
func addExtensionEnvVars(env map[string]string, p provider.Provider, cfg *provider.Config) {
	extensionEnvVars := p.GetExtensionEnvVars(cfg.ImageName)
	for _, varSpec := range extensionEnvVars {
		varName, defaultValue := parseEnvVarSpec(varSpec)
		if value := os.Getenv(varName); value != "" {
			// Host has the var set, use it
			env[varName] = value
		} else if defaultValue != "" {
			// Use the default value from config
			env[varName] = defaultValue
		}
	}

	// Run credential scripts for active extensions
	addCredentialScriptEnvVars(env, cfg)
}

// parseEnvVarSpec parses an env var specification that can be either:
// - "VAR_NAME" - just the variable name (pass-through from host)
// - "VAR_NAME=value" - variable name with default value
func parseEnvVarSpec(spec string) (name, defaultValue string) {
	if idx := strings.Index(spec, "="); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}
	return spec, ""
}

// addCredentialScriptEnvVars runs credential scripts for active extensions
func addCredentialScriptEnvVars(env map[string]string, cfg *provider.Config) {
	// Get the list of extensions being used
	extNames := getActiveExtensionNames(cfg)

	// Load extension configs and run credential scripts
	allExts, err := extensions.GetExtensions()
	if err != nil {
		return
	}

	for _, ext := range allExts {
		if !contains(extNames, ext.Name) {
			continue
		}

		if ext.CredentialScript == "" {
			continue
		}

		// Run the credential script
		credEnvVars, err := extensions.RunCredentialScript(&ext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: credential script for %s failed: %v\n", ext.Name, err)
			continue
		}

		// Add credential env vars (don't override existing values)
		for k, v := range credEnvVars {
			if _, exists := env[k]; !exists {
				env[k] = v
			}
		}
	}
}

// getActiveExtensionNames returns the list of active extension names
func getActiveExtensionNames(cfg *provider.Config) []string {
	if cfg.Extensions == "" {
		// Default to claude if no extensions specified
		return []string{"claude"}
	}
	return strings.Split(cfg.Extensions, ",")
}

// contains checks if a slice contains a value
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if strings.TrimSpace(s) == val {
			return true
		}
	}
	return false
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

// addOtelEnvVars adds OpenTelemetry environment variables
func addOtelEnvVars(env map[string]string, cfg *provider.Config) {
	// Build resource attributes from runtime context
	project := ""
	if cfg.Workdir != "" {
		project = filepath.Base(cfg.Workdir)
	} else if cwd, err := os.Getwd(); err == nil {
		project = filepath.Base(cwd)
	}

	attrs := otel.ResourceAttrs{
		Extension: cfg.Extensions,
		Provider:  cfg.Provider,
		Version:   cfg.AddtVersion,
		Project:   project,
	}

	otelEnvVars := otel.GetEnvVars(cfg.Otel, attrs)
	for k, v := range otelEnvVars {
		env[k] = v
	}
}
