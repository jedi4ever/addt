package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/jedi4ever/addt/config/security"
)

// LoadConfig loads configuration with precedence: defaults < global config < project config < env vars
func LoadConfig(addtVersion, defaultNodeVersion, defaultGoVersion, defaultUvVersion string, defaultPortRangeStart int) *Config {
	// Load config files (project config overrides global config)
	globalCfg := loadGlobalConfig()
	projectCfg := loadProjectConfig()

	// Start with defaults, then apply global config, then project config, then env vars
	cfg := &Config{
		AddtVersion:        addtVersion,
		ExtensionVersions:  make(map[string]string),
		ExtensionAutomount: make(map[string]bool),
	}

	// Node version: default -> global -> project -> env
	cfg.NodeVersion = defaultNodeVersion
	if globalCfg.NodeVersion != "" {
		cfg.NodeVersion = globalCfg.NodeVersion
	}
	if projectCfg.NodeVersion != "" {
		cfg.NodeVersion = projectCfg.NodeVersion
	}
	if v := os.Getenv("ADDT_NODE_VERSION"); v != "" {
		cfg.NodeVersion = v
	}

	// Go version: default -> global -> project -> env
	cfg.GoVersion = defaultGoVersion
	if globalCfg.GoVersion != "" {
		cfg.GoVersion = globalCfg.GoVersion
	}
	if projectCfg.GoVersion != "" {
		cfg.GoVersion = projectCfg.GoVersion
	}
	if v := os.Getenv("ADDT_GO_VERSION"); v != "" {
		cfg.GoVersion = v
	}

	// UV version: default -> global -> project -> env
	cfg.UvVersion = defaultUvVersion
	if globalCfg.UvVersion != "" {
		cfg.UvVersion = globalCfg.UvVersion
	}
	if projectCfg.UvVersion != "" {
		cfg.UvVersion = projectCfg.UvVersion
	}
	if v := os.Getenv("ADDT_UV_VERSION"); v != "" {
		cfg.UvVersion = v
	}

	// Port range start: default -> global -> project -> env
	cfg.PortRangeStart = defaultPortRangeStart
	if globalCfg.PortRangeStart != nil {
		cfg.PortRangeStart = *globalCfg.PortRangeStart
	}
	if projectCfg.PortRangeStart != nil {
		cfg.PortRangeStart = *projectCfg.PortRangeStart
	}
	if v := os.Getenv("ADDT_PORT_RANGE_START"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.PortRangeStart = i
		}
	}

	// SSH forward: default (proxy) -> global -> project -> env
	cfg.SSHForward = "proxy"
	if globalCfg.SSHForward != "" {
		cfg.SSHForward = globalCfg.SSHForward
	}
	if projectCfg.SSHForward != "" {
		cfg.SSHForward = projectCfg.SSHForward
	}
	if v := os.Getenv("ADDT_SSH_FORWARD"); v != "" {
		cfg.SSHForward = v
	}

	// SSH allowed keys: global -> project -> env
	cfg.SSHAllowedKeys = globalCfg.SSHAllowedKeys
	if len(projectCfg.SSHAllowedKeys) > 0 {
		cfg.SSHAllowedKeys = projectCfg.SSHAllowedKeys
	}
	if v := os.Getenv("ADDT_SSH_ALLOWED_KEYS"); v != "" {
		cfg.SSHAllowedKeys = strings.Split(v, ",")
	}

	// Tmux forward: default (false) -> global -> project -> env
	cfg.TmuxForward = false
	if globalCfg.TmuxForward != nil {
		cfg.TmuxForward = *globalCfg.TmuxForward
	}
	if projectCfg.TmuxForward != nil {
		cfg.TmuxForward = *projectCfg.TmuxForward
	}
	if v := os.Getenv("ADDT_TMUX_FORWARD"); v != "" {
		cfg.TmuxForward = v == "true"
	}

	// GPG forward: default (off) -> global -> project -> env
	cfg.GPGForward = ""
	if globalCfg.GPGForward != "" {
		cfg.GPGForward = globalCfg.GPGForward
	}
	if projectCfg.GPGForward != "" {
		cfg.GPGForward = projectCfg.GPGForward
	}
	if v := os.Getenv("ADDT_GPG_FORWARD"); v != "" {
		// Support legacy boolean values
		if v == "true" {
			cfg.GPGForward = "keys"
		} else if v == "false" {
			cfg.GPGForward = ""
		} else {
			cfg.GPGForward = v
		}
	}

	// GPG allowed key IDs: global -> project -> env
	cfg.GPGAllowedKeyIDs = globalCfg.GPGAllowedKeyIDs
	if len(projectCfg.GPGAllowedKeyIDs) > 0 {
		cfg.GPGAllowedKeyIDs = projectCfg.GPGAllowedKeyIDs
	}
	if v := os.Getenv("ADDT_GPG_ALLOWED_KEY_IDS"); v != "" {
		cfg.GPGAllowedKeyIDs = strings.Split(v, ",")
	}

	// DinD mode: default -> global -> project -> env
	cfg.DindMode = globalCfg.DindMode
	if projectCfg.DindMode != "" {
		cfg.DindMode = projectCfg.DindMode
	}
	if v := os.Getenv("ADDT_DIND_MODE"); v != "" {
		cfg.DindMode = v
	}

	// Log enabled: default (false) -> global -> project -> env
	cfg.LogEnabled = false
	if globalCfg.Log != nil {
		cfg.LogEnabled = *globalCfg.Log
	}
	if projectCfg.Log != nil {
		cfg.LogEnabled = *projectCfg.Log
	}
	if v := os.Getenv("ADDT_LOG"); v != "" {
		cfg.LogEnabled = v == "true"
	}

	// Log file: default -> global -> project -> env
	cfg.LogFile = "addt.log"
	if globalCfg.LogFile != "" {
		cfg.LogFile = globalCfg.LogFile
	}
	if projectCfg.LogFile != "" {
		cfg.LogFile = projectCfg.LogFile
	}
	if v := os.Getenv("ADDT_LOG_FILE"); v != "" {
		cfg.LogFile = v
	}

	// Persistent: default (false) -> global -> project -> env
	cfg.Persistent = false
	if globalCfg.Persistent != nil {
		cfg.Persistent = *globalCfg.Persistent
	}
	if projectCfg.Persistent != nil {
		cfg.Persistent = *projectCfg.Persistent
	}
	if v := os.Getenv("ADDT_PERSISTENT"); v != "" {
		cfg.Persistent = v == "true"
	}

	// Workdir automount: default (true) -> global -> project -> env
	cfg.WorkdirAutomount = true
	if globalCfg.WorkdirAutomount != nil {
		cfg.WorkdirAutomount = *globalCfg.WorkdirAutomount
	}
	if projectCfg.WorkdirAutomount != nil {
		cfg.WorkdirAutomount = *projectCfg.WorkdirAutomount
	}
	if v := os.Getenv("ADDT_WORKDIR_AUTOMOUNT"); v != "" {
		cfg.WorkdirAutomount = v != "false"
	}

	// Workdir readonly: default (false) -> global -> project -> env
	cfg.WorkdirReadonly = false
	if globalCfg.WorkdirReadonly != nil {
		cfg.WorkdirReadonly = *globalCfg.WorkdirReadonly
	}
	if projectCfg.WorkdirReadonly != nil {
		cfg.WorkdirReadonly = *projectCfg.WorkdirReadonly
	}
	if v := os.Getenv("ADDT_WORKDIR_READONLY"); v != "" {
		cfg.WorkdirReadonly = v == "true"
	}

	// Firewall: default (false) -> global -> project -> env
	cfg.FirewallEnabled = false
	if globalCfg.Firewall != nil {
		cfg.FirewallEnabled = *globalCfg.Firewall
	}
	if projectCfg.Firewall != nil {
		cfg.FirewallEnabled = *projectCfg.Firewall
	}
	if v := os.Getenv("ADDT_FIREWALL"); v != "" {
		cfg.FirewallEnabled = v == "true"
	}

	// Firewall mode: default (strict) -> global -> project -> env
	cfg.FirewallMode = "strict"
	if globalCfg.FirewallMode != "" {
		cfg.FirewallMode = globalCfg.FirewallMode
	}
	if projectCfg.FirewallMode != "" {
		cfg.FirewallMode = projectCfg.FirewallMode
	}
	if v := os.Getenv("ADDT_FIREWALL_MODE"); v != "" {
		cfg.FirewallMode = v
	}

	// Firewall rules: keep each layer separate for layered override evaluation
	// Order: Defaults → Extension → Global → Project (project wins)
	cfg.GlobalFirewallAllowed = globalCfg.FirewallAllowed
	cfg.GlobalFirewallDenied = globalCfg.FirewallDenied
	cfg.ProjectFirewallAllowed = projectCfg.FirewallAllowed
	cfg.ProjectFirewallDenied = projectCfg.FirewallDenied
	// Extension firewall rules are loaded below after determining the extension

	// GitHub detect: default (false) -> global -> project -> env
	cfg.GitHubDetect = false
	if globalCfg.GitHubDetect != nil {
		cfg.GitHubDetect = *globalCfg.GitHubDetect
	}
	if projectCfg.GitHubDetect != nil {
		cfg.GitHubDetect = *projectCfg.GitHubDetect
	}
	if v := os.Getenv("ADDT_GITHUB_DETECT"); v != "" {
		cfg.GitHubDetect = v == "true"
	}

	// CPUs: default (2) -> global -> project -> env
	cfg.CPUs = "2" // Secure default: limit CPU usage
	if globalCfg.DockerCPUs != "" {
		cfg.CPUs = globalCfg.DockerCPUs
	}
	if projectCfg.DockerCPUs != "" {
		cfg.CPUs = projectCfg.DockerCPUs
	}
	if v := os.Getenv("ADDT_DOCKER_CPUS"); v != "" {
		cfg.CPUs = v
	}

	// Memory: default (4g) -> global -> project -> env
	cfg.Memory = "4g" // Secure default: limit memory usage
	if globalCfg.DockerMemory != "" {
		cfg.Memory = globalCfg.DockerMemory
	}
	if projectCfg.DockerMemory != "" {
		cfg.Memory = projectCfg.DockerMemory
	}
	if v := os.Getenv("ADDT_DOCKER_MEMORY"); v != "" {
		cfg.Memory = v
	}

	// Workdir: default (empty = current dir) -> global -> project -> env
	cfg.Workdir = globalCfg.Workdir
	if projectCfg.Workdir != "" {
		cfg.Workdir = projectCfg.Workdir
	}
	if v := os.Getenv("ADDT_WORKDIR"); v != "" {
		cfg.Workdir = v
	}

	// These don't have global config equivalents
	cfg.EnvVars = strings.Split(getEnvOrDefault("ADDT_ENV_VARS", "ANTHROPIC_API_KEY,GH_TOKEN"), ",")
	cfg.EnvFile = os.Getenv("ADDT_ENV_FILE")
	cfg.Mode = getEnvOrDefault("ADDT_MODE", "container")
	// Auto-detect container runtime (Docker > Podman) if not explicitly set
	cfg.Provider = DetectContainerRuntime()
	cfg.Extensions = os.Getenv("ADDT_EXTENSIONS")
	cfg.Command = os.Getenv("ADDT_COMMAND")

	// Load per-extension config from config files
	// Precedence: global config < project config < environment variables
	if globalCfg.Extensions != nil {
		for extName, extCfg := range globalCfg.Extensions {
			if extCfg.Version != "" {
				cfg.ExtensionVersions[extName] = extCfg.Version
			}
			if extCfg.Automount != nil {
				cfg.ExtensionAutomount[extName] = *extCfg.Automount
			}
		}
	}
	if projectCfg.Extensions != nil {
		for extName, extCfg := range projectCfg.Extensions {
			if extCfg.Version != "" {
				cfg.ExtensionVersions[extName] = extCfg.Version
			}
			if extCfg.Automount != nil {
				cfg.ExtensionAutomount[extName] = *extCfg.Automount
			}
		}
	}

	// Load extension-specific firewall rules based on ADDT_EXTENSIONS
	// Extension firewall rules are stored in global config under extensions.<name>
	currentExt := os.Getenv("ADDT_EXTENSIONS")
	if currentExt != "" {
		// Use first extension if multiple specified
		extName := strings.Split(currentExt, ",")[0]
		if globalCfg.Extensions != nil && globalCfg.Extensions[extName] != nil {
			extCfg := globalCfg.Extensions[extName]
			cfg.ExtensionFirewallAllowed = extCfg.FirewallAllowed
			cfg.ExtensionFirewallDenied = extCfg.FirewallDenied
		}
	}

	// Load per-extension versions and mount configs from environment (overrides config files)
	// Pattern: ADDT_<EXT>_VERSION and ADDT_<EXT>_AUTOMOUNT
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		// Check for ADDT_<EXT>_VERSION pattern
		if strings.HasPrefix(key, "ADDT_") && strings.HasSuffix(key, "_VERSION") {
			// Extract extension name (e.g., "ADDT_CLAUDE_VERSION" -> "claude")
			extName := strings.TrimPrefix(key, "ADDT_")
			extName = strings.TrimSuffix(extName, "_VERSION")
			extName = strings.ToLower(extName)
			// Skip non-extension versions (node, go, uv)
			if extName != "node" && extName != "go" && extName != "uv" {
				cfg.ExtensionVersions[extName] = value
			}
		}

		// Check for ADDT_<EXT>_AUTOMOUNT pattern
		if strings.HasPrefix(key, "ADDT_") && strings.HasSuffix(key, "_AUTOMOUNT") {
			// Extract extension name (e.g., "ADDT_CLAUDE_AUTOMOUNT" -> "claude")
			extName := strings.TrimPrefix(key, "ADDT_")
			extName = strings.TrimSuffix(extName, "_AUTOMOUNT")
			extName = strings.ToLower(extName)
			cfg.ExtensionAutomount[extName] = value != "false"
		}
	}

	// Set default version for claude if not specified
	if _, exists := cfg.ExtensionVersions["claude"]; !exists {
		cfg.ExtensionVersions["claude"] = "stable"
	}

	// Parse ports
	if ports := os.Getenv("ADDT_PORTS"); ports != "" {
		cfg.Ports = strings.Split(ports, ",")
		for i := range cfg.Ports {
			cfg.Ports[i] = strings.TrimSpace(cfg.Ports[i])
		}
	}

	// Trim env vars
	for i := range cfg.EnvVars {
		cfg.EnvVars[i] = strings.TrimSpace(cfg.EnvVars[i])
	}

	// Load security configuration using the security package
	cfg.Security = security.LoadConfig(globalCfg.Security, projectCfg.Security)

	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// mergeStringSlices merges two string slices, removing duplicates
func mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
