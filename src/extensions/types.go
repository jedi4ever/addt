package extensions

// ExtensionMount represents a mount configuration for an extension
type ExtensionMount struct {
	Source string `yaml:"source" json:"source"`
	Target string `yaml:"target" json:"target"`
}

// ExtensionFlag represents a CLI flag for an extension
type ExtensionFlag struct {
	Flag        string `yaml:"flag" json:"flag"`
	Description string `yaml:"description" json:"description"`
}

// ExtensionConfig represents the config.yaml structure for extension source files
// Used when reading extension configs from embedded filesystem or local ~/.addt/extensions/
type ExtensionConfig struct {
	Name           string           `yaml:"name" json:"name"`
	Description    string           `yaml:"description" json:"description"`
	Entrypoint     string           `yaml:"entrypoint" json:"entrypoint"`
	DefaultVersion string           `yaml:"default_version" json:"default_version,omitempty"`
	AutoMount      bool             `yaml:"auto_mount" json:"auto_mount"`
	Dependencies   []string         `yaml:"dependencies" json:"dependencies,omitempty"`
	EnvVars        []string         `yaml:"env_vars" json:"env_vars,omitempty"`
	Mounts         []ExtensionMount `yaml:"mounts" json:"mounts,omitempty"`
	Flags          []ExtensionFlag  `yaml:"flags" json:"flags,omitempty"`
	IsLocal        bool             `yaml:"-" json:"-"` // Runtime flag, not serialized
}

// ExtensionMetadata represents metadata for an installed extension inside a Docker image
// Used when reading extensions.json from built Docker images
type ExtensionMetadata struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Entrypoint  string           `json:"entrypoint"`
	AutoMount   *bool            `json:"auto_mount,omitempty"` // nil or true = auto mount, false = only if explicitly enabled
	Mounts      []ExtensionMount `json:"mounts,omitempty"`
	Flags       []ExtensionFlag  `json:"flags,omitempty"`
	EnvVars     []string         `json:"env_vars,omitempty"`
}

// ExtensionsJSONConfig represents the extensions.json file structure inside Docker images
type ExtensionsJSONConfig struct {
	Extensions map[string]ExtensionMetadata `json:"extensions"`
}

// ExtensionMountWithName includes the extension name for mount filtering
type ExtensionMountWithName struct {
	Source        string
	Target        string
	ExtensionName string
	AutoMount     *bool // from extension level
}
