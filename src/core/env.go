package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedi4ever/addt/config/otel"
	"github.com/jedi4ever/addt/extensions"
	"github.com/jedi4ever/addt/provider"
	"github.com/jedi4ever/addt/util"
	"github.com/jedi4ever/addt/util/terminal"
)

var envLogger = util.Log("env")

// BuildEnvironment creates the environment variables map for the container
func BuildEnvironment(p provider.Provider, cfg *provider.Config) map[string]string {
	env := make(map[string]string)

	// Add extension-required environment variables
	addExtensionEnvVars(env, p, cfg)

	// Add user-configured environment variables
	addUserEnvVars(env, cfg)

	// Add terminal environment variables
	addTerminalEnvVars(env)

	// Inject port mapping info into system prompt
	PortsInjectPrompt(env, cfg)

	// Add firewall configuration
	addFirewallEnvVars(env, cfg)

	// Add GitHub scope configuration
	addGitHubScopeEnvVars(env, cfg)

	// Add command override
	addCommandEnvVar(env, cfg)

	// Add logging configuration
	addLoggingEnvVars(env)

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
	// Track credential var names so the entrypoint can unset them after setup
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

// addCredentialScriptEnvVars runs credential scripts for active extensions.
// It also sets ADDT_CREDENTIAL_VARS with the list of credential env var names
// so the entrypoint can unset them after setup (preventing leaks into shell sessions).
func addCredentialScriptEnvVars(env map[string]string, cfg *provider.Config) {
	// Get the list of extensions being used
	extNames := getActiveExtensionNames(cfg)

	// Load extension configs and run credential scripts
	allExts, err := extensions.GetExtensions()
	if err != nil {
		return
	}

	var credVarNames []string

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
			envLogger.Warning("credential script for %s failed: %v", ext.Name, err)
			continue
		}

		// Add credential env vars (don't override existing values)
		for k, v := range credEnvVars {
			if _, exists := env[k]; !exists {
				env[k] = v
			}
			credVarNames = append(credVarNames, k)
		}
	}

	// Tell the entrypoint which env vars to unset after setup
	if len(credVarNames) > 0 {
		env["ADDT_CREDENTIAL_VARS"] = strings.Join(credVarNames, ",")
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

// addFlagEnvVars sets env vars for flags from CLI args and config settings.
// Precedence: CLI flags > config settings (config settings fill in the rest).
func addFlagEnvVars(env map[string]string, cfg *provider.Config, args []string) {
	extNames := getActiveExtensionNames(cfg)

	allExts, err := extensions.GetExtensions()
	if err != nil {
		return
	}

	for _, ext := range allExts {
		if !contains(extNames, ext.Name) {
			continue
		}

		for _, flag := range ext.Flags {
			if flag.EnvVar == "" {
				continue
			}

			flagKey := strings.TrimPrefix(flag.Flag, "--")

			// Check CLI args first (highest precedence)
			cliSet := false
			for _, arg := range args {
				if arg == flag.Flag {
					env[flag.EnvVar] = "true"
					envLogger.Debugf("Flag %s (CLI) sets %s=true", flag.Flag, flag.EnvVar)
					cliSet = true
					break
				}
			}

			// If not set by CLI, check config settings
			if !cliSet {
				if cfg.ExtensionFlagSettings != nil {
					if flagSettings, ok := cfg.ExtensionFlagSettings[ext.Name]; ok {
						if val, ok := flagSettings[flagKey]; ok && val {
							env[flag.EnvVar] = "true"
							envLogger.Debugf("Flag %s (config) sets %s=true", flag.Flag, flag.EnvVar)
						}
					}
				}
			}
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

// addGitHubScopeEnvVars adds GitHub token scoping environment variables
func addGitHubScopeEnvVars(env map[string]string, cfg *provider.Config) {
	if cfg.GitHubScopeToken {
		env["ADDT_GITHUB_SCOPE_TOKEN"] = "true"
	}
	if len(cfg.GitHubScopeRepos) > 0 {
		env["ADDT_GITHUB_SCOPE_REPOS"] = strings.Join(cfg.GitHubScopeRepos, ",")
	}
}

// addLoggingEnvVars adds logging-related environment variables
func addLoggingEnvVars(env map[string]string) {
	// Pass ADDT_LOG_LEVEL to container if set
	if logLevel := os.Getenv("ADDT_LOG_LEVEL"); logLevel != "" {
		env["ADDT_LOG_LEVEL"] = logLevel
		envLogger.Debugf("Passing ADDT_LOG_LEVEL=%s to container", logLevel)
	}
	// Pass ADDT_LOG_FILE to container if set
	if logFile := os.Getenv("ADDT_LOG_FILE"); logFile != "" {
		env["ADDT_LOG_FILE"] = logFile
		envLogger.Debugf("Passing ADDT_LOG_FILE=%s to container", logFile)
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
