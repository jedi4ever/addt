package config

import (
	"github.com/jedi4ever/addt/config/otel"
	"github.com/jedi4ever/addt/config/security"
)

// ExtensionSettings holds per-extension configuration settings
type ExtensionSettings struct {
	Version         string           `yaml:"version,omitempty"`
	Automount       *bool            `yaml:"automount,omitempty"`
	FirewallAllowed []string         `yaml:"firewall_allowed,omitempty"`
	FirewallDenied  []string         `yaml:"firewall_denied,omitempty"`
	Flags           map[string]*bool `yaml:"flags,omitempty"`
}

// DindSettings holds Docker-in-Docker configuration
type DindSettings struct {
	Enable *bool  `yaml:"enable,omitempty"`
	Mode   string `yaml:"mode,omitempty"`
}

// DockerSettings holds Docker-specific configuration (DinD)
type DockerSettings struct {
	Dind *DindSettings `yaml:"dind,omitempty"`
}

// ContainerSettings holds container resource limits
type ContainerSettings struct {
	CPUs   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// VmSettings holds VM resource configuration (Podman machine, Docker Desktop)
type VmSettings struct {
	CPUs   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// PortsSettings holds port forwarding configuration
type PortsSettings struct {
	Forward            *bool    `yaml:"forward,omitempty"`
	Expose             []string `yaml:"expose,omitempty"`
	RangeStart         *int     `yaml:"range_start,omitempty"`
	InjectSystemPrompt *bool    `yaml:"inject_system_prompt,omitempty"`
}

// SSHSettings holds SSH forwarding configuration
type SSHSettings struct {
	ForwardKeys *bool    `yaml:"forward_keys,omitempty"`
	ForwardMode string   `yaml:"forward_mode,omitempty"`
	AllowedKeys []string `yaml:"allowed_keys,omitempty"`
}

// GitHubSettings holds GitHub token forwarding configuration
type GitHubSettings struct {
	ForwardToken *bool  `yaml:"forward_token,omitempty"`
	TokenSource  string `yaml:"token_source,omitempty"`
}

// FirewallSettings holds network firewall configuration
type FirewallSettings struct {
	Enabled *bool    `yaml:"enabled,omitempty"`
	Mode    string   `yaml:"mode,omitempty"`
	Allowed []string `yaml:"allowed,omitempty"`
	Denied  []string `yaml:"denied,omitempty"`
}

// GPGSettings holds GPG forwarding configuration
type GPGSettings struct {
	Forward       string   `yaml:"forward,omitempty"`         // "proxy", "agent", "keys", or "off"
	AllowedKeyIDs []string `yaml:"allowed_key_ids,omitempty"` // GPG key IDs allowed
}

// LogSettings holds logging configuration
type LogSettings struct {
	Enabled  *bool  `yaml:"enabled,omitempty"`   // Enable command logging
	Output   string `yaml:"output,omitempty"`    // Output target: stderr, stdout, file (default: stderr)
	File     string `yaml:"file,omitempty"`      // Log file name (default: addt.log)
	Dir      string `yaml:"dir,omitempty"`       // Log directory (default: ~/.addt/logs)
	Level    string `yaml:"level,omitempty"`     // Log level: DEBUG, INFO, WARN, ERROR (default: INFO)
	Modules  string `yaml:"modules,omitempty"`   // Comma-separated module filter (default: * for all)
	Rotate   *bool  `yaml:"rotate,omitempty"`    // Enable log rotation (default: false)
	MaxSize  string `yaml:"max_size,omitempty"`  // Max file size before rotating (e.g. "10m", default: 10m)
	MaxFiles *int   `yaml:"max_files,omitempty"` // Number of rotated files to keep (default: 5)
}

// WorkdirSettings holds working directory configuration
type WorkdirSettings struct {
	Path      string `yaml:"path,omitempty"`      // Override working directory (default: current directory)
	Automount *bool  `yaml:"automount,omitempty"` // Auto-mount working directory to /workspace
	Readonly  *bool  `yaml:"readonly,omitempty"`  // Mount working directory as read-only
}

// GlobalConfig represents the persistent configuration stored in ~/.addt/config.yaml
type GlobalConfig struct {
	Container        *ContainerSettings `yaml:"container,omitempty"`
	Docker           *DockerSettings    `yaml:"docker,omitempty"`
	Vm               *VmSettings        `yaml:"vm,omitempty"`
	Firewall         *FirewallSettings `yaml:"firewall,omitempty"`
	GitHub           *GitHubSettings `yaml:"github,omitempty"`
	EnvFileLoad      *bool           `yaml:"env_file_load,omitempty"`
	EnvFile          string          `yaml:"env_file,omitempty"`
	GoVersion        string          `yaml:"go_version,omitempty"`
	GPG              *GPGSettings    `yaml:"gpg,omitempty"`
	Log              *LogSettings    `yaml:"log,omitempty"`
	NodeVersion      string          `yaml:"node_version,omitempty"`
	Persistent       *bool           `yaml:"persistent,omitempty"`
	Ports            *PortsSettings  `yaml:"ports,omitempty"`
	SSH              *SSHSettings    `yaml:"ssh,omitempty"`
	TmuxForward      *bool           `yaml:"tmux_forward,omitempty"`
	HistoryPersist   *bool           `yaml:"history_persist,omitempty"` // Persist shell history between sessions
	UvVersion        string          `yaml:"uv_version,omitempty"`
	Workdir          *WorkdirSettings `yaml:"workdir,omitempty"`

	// Per-extension configuration
	Extensions map[string]*ExtensionSettings `yaml:"extensions,omitempty"`

	// Security configuration
	Security *security.Settings `yaml:"security,omitempty"`

	// OpenTelemetry configuration
	Otel *otel.Settings `yaml:"otel,omitempty"`
}

// Config holds all configuration options
type Config struct {
	AddtVersion              string
	NodeVersion              string
	GoVersion                string
	UvVersion                string
	EnvVars                  []string
	GitHubForwardToken       bool
	GitHubTokenSource        string
	Ports                    []string
	PortRangeStart           int
	PortsInjectSystemPrompt  bool
	SSHForwardKeys           bool
	SSHForwardMode           string
	SSHAllowedKeys           []string
	TmuxForward              bool
	HistoryPersist           bool     // Persist shell history between sessions (default: false)
	GPGForward               string   // "proxy", "agent", "keys", or "off"
	GPGAllowedKeyIDs         []string // GPG key IDs allowed for signing
	DockerDindMode           string
	EnvFileLoad              bool
	EnvFile                  string
	LogEnabled               bool
	LogOutput                string // stderr, stdout, file (default: stderr)
	LogFile                  string
	LogLevel                 string // DEBUG, INFO, WARN, ERROR (default: INFO)
	LogDir                   string // Log directory (default: ~/.addt/logs)
	LogModules               string // Comma-separated module filter (default: * for all)
	LogRotate                bool   // Enable log rotation
	LogMaxSize               string // Max file size before rotating (e.g. "10m")
	LogMaxFiles              int    // Number of rotated files to keep
	ImageName                string
	Persistent               bool                       // Enable persistent container mode
	WorkdirAutomount         bool                       // Auto-mount working directory
	WorkdirReadonly          bool                       // Mount working directory as read-only
	Workdir                  string                     // Override working directory (default: current directory)
	FirewallEnabled          bool                       // Enable network firewall
	FirewallMode             string                     // Firewall mode: strict, permissive, off
	GlobalFirewallAllowed    []string                   // Global allowed domains
	GlobalFirewallDenied     []string                   // Global denied domains
	ProjectFirewallAllowed   []string                   // Project allowed domains
	ProjectFirewallDenied    []string                   // Project denied domains
	ExtensionFirewallAllowed []string                   // Extension allowed domains
	ExtensionFirewallDenied  []string                   // Extension denied domains
	Mode                     string                     // container or shell
	Provider                 string                     // Provider type: docker or daytona
	Extensions               string                     // Comma-separated list of extensions to install (e.g., "claude,codex")
	Command                  string                     // Command to run instead of claude (e.g., "gt" for gastown)
	ExtensionVersions        map[string]string          // Per-extension versions (e.g., {"claude": "1.0.5", "codex": "latest"})
	ExtensionAutomount       map[string]bool            // Per-extension automount control (e.g., {"claude": true, "codex": false})
	ExtensionFlagSettings    map[string]map[string]bool // Per-extension flag settings from config (e.g., {"claude": {"yolo": true}})
	ContainerCPUs            string                     // Container CPU limit (e.g., "2", "0.5", "1.5")
	ContainerMemory          string                     // Container memory limit (e.g., "512m", "2g", "4gb")

	// Security settings
	Security security.Config

	// OpenTelemetry settings
	Otel otel.Config
}
