package provider

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
	Persistent         bool
	WorkdirAutomount   bool
	Workdir            string
	FirewallEnabled    bool
	FirewallMode       string
	Mode               string
	Provider           string
	Extensions         string
	Command            string
	ExtensionVersions  map[string]string // Per-extension versions (e.g., {"claude": "1.0.5", "codex": "latest"})
	ExtensionAutomount map[string]bool   // Per-extension automount control (e.g., {"claude": true, "codex": false})
	NoCache            bool              // Disable Docker cache for builds
	CPUs               string            // CPU limit (e.g., "2", "0.5", "1.5")
	Memory             string            // Memory limit (e.g., "512m", "2g", "4gb")
}

// RunSpec specifies how to run a container/workspace
type RunSpec struct {
	Name        string
	ImageName   string
	Args        []string
	WorkDir     string
	Interactive bool
	Persistent  bool
	Volumes     []VolumeMount
	Ports       []PortMapping
	Env         map[string]string
	SSHForward  string
	GPGForward  bool
	DindMode    string
	CPUs        string // CPU limit (e.g., "2", "0.5")
	Memory      string // Memory limit (e.g., "512m", "2g")
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
