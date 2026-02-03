package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtensionSettings holds per-extension configuration settings
type ExtensionSettings struct {
	Version   string `yaml:"version,omitempty"`
	Automount *bool  `yaml:"automount,omitempty"`
}

// GlobalConfig represents the persistent configuration stored in ~/.addt/config.yaml
type GlobalConfig struct {
	Dind             *bool  `yaml:"dind,omitempty"`
	DindMode         string `yaml:"dind_mode,omitempty"`
	DockerCPUs       string `yaml:"docker_cpus,omitempty"`
	DockerMemory     string `yaml:"docker_memory,omitempty"`
	Firewall         *bool  `yaml:"firewall,omitempty"`
	FirewallMode     string `yaml:"firewall_mode,omitempty"`
	GitHubDetect     *bool  `yaml:"github_detect,omitempty"`
	GoVersion        string `yaml:"go_version,omitempty"`
	GPGForward       *bool  `yaml:"gpg_forward,omitempty"`
	Log              *bool  `yaml:"log,omitempty"`
	LogFile          string `yaml:"log_file,omitempty"`
	NodeVersion      string `yaml:"node_version,omitempty"`
	Persistent       *bool  `yaml:"persistent,omitempty"`
	PortRangeStart   *int   `yaml:"port_range_start,omitempty"`
	SSHForward       string `yaml:"ssh_forward,omitempty"`
	UvVersion        string `yaml:"uv_version,omitempty"`
	Workdir          string `yaml:"workdir,omitempty"`
	WorkdirAutomount *bool  `yaml:"workdir_automount,omitempty"`

	// Per-extension configuration
	Extensions map[string]*ExtensionSettings `yaml:"extensions,omitempty"`
}

// loadGlobalConfig loads the global config from ~/.addt/config.yaml
// Can be overridden with ADDT_CONFIG_DIR environment variable
func loadGlobalConfig() *GlobalConfig {
	configDir := os.Getenv("ADDT_CONFIG_DIR")
	if configDir == "" {
		currentUser, err := user.Current()
		if err != nil {
			return &GlobalConfig{}
		}
		configDir = filepath.Join(currentUser.HomeDir, ".addt")
	}

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &GlobalConfig{}
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return &GlobalConfig{}
	}

	return &cfg
}

// loadProjectConfig loads the project config from .addt.yaml in current directory
func loadProjectConfig() *GlobalConfig {
	cwd, err := os.Getwd()
	if err != nil {
		return &GlobalConfig{}
	}

	configPath := filepath.Join(cwd, ".addt.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &GlobalConfig{}
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return &GlobalConfig{}
	}

	return &cfg
}

// Config holds all configuration options
type Config struct {
	NodeVersion        string
	GoVersion          string
	UvVersion          string
	EnvVars            []string
	GitHubDetect       bool
	Ports              []string
	PortRangeStart     int
	SSHForward         string
	GPGForward         bool
	DindMode           string
	EnvFile            string
	LogEnabled         bool
	LogFile            string
	ImageName          string
	Persistent         bool              // Enable persistent container mode
	WorkdirAutomount   bool              // Auto-mount working directory
	Workdir            string            // Override working directory (default: current directory)
	FirewallEnabled    bool              // Enable network firewall
	FirewallMode       string            // Firewall mode: strict, permissive, off
	Mode               string            // container or shell
	Provider           string            // Provider type: docker or daytona
	Extensions         string            // Comma-separated list of extensions to install (e.g., "gastown,beads")
	Command            string            // Command to run instead of claude (e.g., "gt" for gastown)
	ExtensionVersions  map[string]string // Per-extension versions (e.g., {"claude": "1.0.5", "codex": "latest"})
	ExtensionAutomount map[string]bool   // Per-extension automount control (e.g., {"claude": true, "codex": false})
	CPUs               string            // CPU limit (e.g., "2", "0.5", "1.5")
	Memory             string            // Memory limit (e.g., "512m", "2g", "4gb")
}

// LoadConfig loads configuration with precedence: defaults < global config < project config < env vars
func LoadConfig(defaultNodeVersion string, defaultGoVersion string, defaultUvVersion string, defaultPortRangeStart int) *Config {
	// Load config files (project config overrides global config)
	globalCfg := loadGlobalConfig()
	projectCfg := loadProjectConfig()

	// Start with defaults, then apply global config, then project config, then env vars
	cfg := &Config{
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

	// SSH forward: default -> global -> project -> env
	cfg.SSHForward = globalCfg.SSHForward
	if projectCfg.SSHForward != "" {
		cfg.SSHForward = projectCfg.SSHForward
	}
	if v := os.Getenv("ADDT_SSH_FORWARD"); v != "" {
		cfg.SSHForward = v
	}

	// GPG forward: default (false) -> global -> project -> env
	cfg.GPGForward = false
	if globalCfg.GPGForward != nil {
		cfg.GPGForward = *globalCfg.GPGForward
	}
	if projectCfg.GPGForward != nil {
		cfg.GPGForward = *projectCfg.GPGForward
	}
	if v := os.Getenv("ADDT_GPG_FORWARD"); v != "" {
		cfg.GPGForward = v == "true"
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

	// CPUs: default (empty) -> global -> project -> env
	cfg.CPUs = globalCfg.DockerCPUs
	if projectCfg.DockerCPUs != "" {
		cfg.CPUs = projectCfg.DockerCPUs
	}
	if v := os.Getenv("ADDT_DOCKER_CPUS"); v != "" {
		cfg.CPUs = v
	}

	// Memory: default (empty) -> global -> project -> env
	cfg.Memory = globalCfg.DockerMemory
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
	cfg.Provider = getEnvOrDefault("ADDT_PROVIDER", "docker")
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

	return cfg
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
