package extensions

import (
	"encoding/json"
)

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

// Entrypoint can be either a string or an array of strings
// Examples:
//
//	entrypoint: claude
//	entrypoint: ["bash", "-i"]
type Entrypoint []string

// UnmarshalYAML handles both string and array formats
func (e *Entrypoint) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try string first
	var str string
	if err := unmarshal(&str); err == nil {
		*e = []string{str}
		return nil
	}

	// Try array
	var arr []string
	if err := unmarshal(&arr); err != nil {
		return err
	}
	*e = arr
	return nil
}

// UnmarshalJSON handles both string and array formats
func (e *Entrypoint) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*e = []string{str}
		return nil
	}

	// Try array
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*e = arr
	return nil
}

// MarshalJSON outputs as array
func (e Entrypoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(e))
}

// Command returns the command (first element)
func (e Entrypoint) Command() string {
	if len(e) == 0 {
		return ""
	}
	return e[0]
}

// Args returns the arguments (elements after the first)
func (e Entrypoint) Args() []string {
	if len(e) <= 1 {
		return []string{}
	}
	return e[1:]
}

// ExtensionConfig represents the config.yaml structure for extension source files
// Used when reading extension configs from embedded filesystem or local ~/.addt/extensions/
type ExtensionConfig struct {
	Name           string           `yaml:"name" json:"name"`
	Description    string           `yaml:"description" json:"description"`
	Entrypoint     Entrypoint       `yaml:"entrypoint" json:"entrypoint"`
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
	Entrypoint  Entrypoint       `json:"entrypoint"`
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
