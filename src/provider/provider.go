package provider

import (
	"github.com/jedi4ever/addt/config/otel"
	"github.com/jedi4ever/addt/config/security"
)

// Provider is the interface for container runtime providers (Docker, Daytona, etc.)
type Provider interface {
	// Core lifecycle
	Initialize(cfg *Config) error
	Run(spec *RunSpec) error
	Shell(spec *RunSpec) error
	Cleanup() error

	// Environment management
	Exists(name string) bool
	IsRunning(name string) bool
	Start(name string) error
	Stop(name string) error
	Remove(name string) error
	List() ([]Environment, error)

	// Environment naming
	GeneratePersistentName() string
	GenerateEphemeralName() string

	// Environment preparation (Docker: builds images, Daytona: no-op)
	BuildIfNeeded(rebuild bool, rebuildBase bool) error
	DetermineImageName() string

	// Status information
	GetStatus(cfg *Config, envName string) string
	GetName() string // "docker" or "daytona"

	// Extension metadata
	GetExtensionEnvVars(imageName string) []string
}

// Config holds provider configuration
type Config struct {
	AddtVersion             string
	NodeVersion             string
	GoVersion               string
	UvVersion               string
	EnvVars                 []string
	GitHubForwardToken      bool
	GitHubTokenSource       string
	Ports                   []string
	PortRangeStart          int
	PortsInjectSystemPrompt bool
	SSHForwardKeys          bool
	SSHForwardMode          string
	SSHAllowedKeys          []string
	TmuxForward             bool
	HistoryPersist          bool
	GPGForward              string   // "proxy", "agent", "keys", or "off"
	GPGAllowedKeyIDs        []string // GPG key IDs (fingerprints) that are allowed
	DockerDindMode          string
	EnvFileLoad             bool
	EnvFile                 string
	LogEnabled              bool
	LogFile                 string
	ImageName               string
	Persistent              bool
	WorkdirAutomount        bool
	WorkdirReadonly         bool
	Workdir                 string
	FirewallEnabled         bool
	FirewallMode            string
	Mode                    string
	Provider                string
	Extensions              string
	Command                 string
	ExtensionVersions       map[string]string          // Per-extension versions (e.g., {"claude": "1.0.5", "codex": "latest"})
	ExtensionAutomount      map[string]bool            // Per-extension automount control (e.g., {"claude": true, "codex": false})
	ExtensionFlagSettings   map[string]map[string]bool // Per-extension flag settings from config (e.g., {"claude": {"yolo": true}})
	NoCache                 bool                       // Disable Docker cache for builds
	ContainerCPUs           string                     // Container CPU limit (e.g., "2", "0.5", "1.5")
	ContainerMemory         string                     // Container memory limit (e.g., "512m", "2g", "4gb")

	// Security settings
	Security security.Config

	// OpenTelemetry settings
	Otel otel.Config
}

// RunSpec specifies how to run a container/workspace
type RunSpec struct {
	Name             string
	ImageName        string
	Args             []string
	WorkDir          string
	Interactive      bool
	Persistent       bool
	Volumes          []VolumeMount
	Ports            []PortMapping
	Env              map[string]string
	SSHForwardKeys   bool
	SSHForwardMode   string
	SSHAllowedKeys   []string
	TmuxForward      bool
	HistoryPersist   bool
	GPGForward       string   // "proxy", "agent", "keys", or "off"
	GPGAllowedKeyIDs []string // GPG key IDs that are allowed
	DockerDindMode   string
	ContainerCPUs    string // Container CPU limit (e.g., "2", "0.5")
	ContainerMemory  string // Container memory limit (e.g., "512m", "2g")
}

// Environment represents a container or workspace
type Environment struct {
	Name      string
	Status    string // "running", "stopped", "exited"
	CreatedAt string
}

// VolumeMount represents a volume mount
type VolumeMount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// PortMapping represents a port mapping
type PortMapping struct {
	Container int
	Host      int
}
