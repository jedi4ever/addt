package config

import (
	"os"
	"strconv"
	"strings"
)

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

// LoadConfig loads configuration from environment variables
func LoadConfig(defaultNodeVersion string, defaultGoVersion string, defaultUvVersion string, defaultPortRangeStart int) *Config {
	cfg := &Config{
		NodeVersion:        getEnvOrDefault("ADDT_NODE_VERSION", defaultNodeVersion),
		GoVersion:          getEnvOrDefault("ADDT_GO_VERSION", defaultGoVersion),
		UvVersion:          getEnvOrDefault("ADDT_UV_VERSION", defaultUvVersion),
		EnvVars:            strings.Split(getEnvOrDefault("ADDT_ENV_VARS", "ANTHROPIC_API_KEY,GH_TOKEN"), ","),
		GitHubDetect:       getEnvOrDefault("ADDT_GITHUB_DETECT", "false") == "true",
		PortRangeStart:     getEnvInt("ADDT_PORT_RANGE_START", defaultPortRangeStart),
		SSHForward:         os.Getenv("ADDT_SSH_FORWARD"),
		GPGForward:         os.Getenv("ADDT_GPG_FORWARD") == "true",
		DindMode:           os.Getenv("ADDT_DIND_MODE"),
		EnvFile:            os.Getenv("ADDT_ENV_FILE"), // Empty means use default .env
		LogEnabled:         os.Getenv("ADDT_LOG") == "true",
		LogFile:            getEnvOrDefault("ADDT_LOG_FILE", "addt.log"),
		Persistent:         os.Getenv("ADDT_PERSISTENT") == "true",
		WorkdirAutomount:   getEnvOrDefault("ADDT_WORKDIR_AUTOMOUNT", "true") != "false",
		Workdir:            os.Getenv("ADDT_WORKDIR"),
		FirewallEnabled:    os.Getenv("ADDT_FIREWALL") == "true",
		FirewallMode:       getEnvOrDefault("ADDT_FIREWALL_MODE", "strict"),
		Mode:               getEnvOrDefault("ADDT_MODE", "container"),
		Provider:           getEnvOrDefault("ADDT_PROVIDER", "docker"),
		Extensions:         os.Getenv("ADDT_EXTENSIONS"), // No default - must be set via symlink or env
		Command:            os.Getenv("ADDT_COMMAND"),
		ExtensionVersions:  make(map[string]string),
		ExtensionAutomount: make(map[string]bool),
		CPUs:               os.Getenv("ADDT_CPUS"),   // CPU limit (e.g., "2", "0.5")
		Memory:             os.Getenv("ADDT_MEMORY"), // Memory limit (e.g., "512m", "2g")
	}

	// Load per-extension versions and mount configs from environment
	// Pattern: ADDT_<EXT>_VERSION and ADDT_MOUNT_<EXT>_CONFIG
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
