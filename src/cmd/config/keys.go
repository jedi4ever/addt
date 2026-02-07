package config

import (
	"fmt"
	"strings"

	cfgtypes "github.com/jedi4ever/addt/config"
	"github.com/jedi4ever/addt/config/otel"
	"github.com/jedi4ever/addt/config/security"
	"github.com/jedi4ever/addt/extensions"
)

// KeyInfo holds metadata about a config key
type KeyInfo struct {
	Key         string
	Description string
	Type        string // "bool", "string", "int"
	EnvVar      string
}

// GetKeys returns all valid config keys with their metadata (sorted alphabetically)
func GetKeys() []KeyInfo {
	keys := []KeyInfo{
		{Key: "env_file_load", Description: "Load .env file (default: true)", Type: "bool", EnvVar: "ADDT_ENV_FILE_LOAD"},
		{Key: "env_file", Description: "Path to .env file (default: .env)", Type: "string", EnvVar: "ADDT_ENV_FILE"},
		{Key: "go_version", Description: "Go version", Type: "string", EnvVar: "ADDT_GO_VERSION"},
		{Key: "node_version", Description: "Node.js version", Type: "string", EnvVar: "ADDT_NODE_VERSION"},
		{Key: "persistent", Description: "Enable persistent container mode", Type: "bool", EnvVar: "ADDT_PERSISTENT"},
		{Key: "history_persist", Description: "Persist shell history between sessions (default: false)", Type: "bool", EnvVar: "ADDT_HISTORY_PERSIST"},
		{Key: "uv_version", Description: "UV Python package manager version", Type: "string", EnvVar: "ADDT_UV_VERSION"},
	}
	// Add firewall keys
	keys = append(keys, GetFirewallKeys()...)
	// Add github keys
	keys = append(keys, GetGitHubKeys()...)
	// Add GPG keys
	keys = append(keys, GetGPGKeys()...)
	// Add log keys
	keys = append(keys, GetLogKeys()...)
	// Add ports keys
	keys = append(keys, GetPortsKeys()...)
	// Add SSH keys
	keys = append(keys, GetSSHKeys()...)
	// Add workdir keys
	keys = append(keys, GetWorkdirKeys()...)
	// Add container keys
	keys = append(keys, GetContainerKeys()...)
	// Add VM keys
	keys = append(keys, GetVmKeys()...)
	// Add docker keys
	keys = append(keys, GetDockerKeys()...)
	// Add security keys
	keys = append(keys, GetSecurityKeys()...)
	// Add OTEL keys
	keys = append(keys, GetOtelKeys()...)
	return keys
}

// GetExtensionKeys returns all valid extension config keys with their metadata
func GetExtensionKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "version", Description: "Extension version", Type: "string", EnvVar: "ADDT_%s_VERSION"},
		{Key: "automount", Description: "Auto-mount extension config directories", Type: "bool", EnvVar: "ADDT_%s_AUTOMOUNT"},
	}
}

// GetDefaultValue returns the default value for a config key
func GetDefaultValue(key string) string {
	switch key {
	case "container.cpus":
		return "2"
	case "container.memory":
		return "4g"
	case "docker.dind.enable":
		return "false"
	case "docker.dind.mode":
		return "isolated"
	case "env_file_load":
		return "true"
	case "env_file":
		return ".env"
	case "firewall.enabled":
		return "false"
	case "firewall.mode":
		return "strict"
	case "github.forward_token":
		return "true"
	case "github.token_source":
		return "gh_auth"
	case "go_version":
		return "latest"
	case "gpg.forward":
		return ""
	case "gpg.allowed_key_ids":
		return ""
	case "log.enabled":
		return "false"
	case "log.output":
		return "stderr"
	case "log.file":
		return "addt.log"
	case "log.dir":
		return ""
	case "log.level":
		return "INFO"
	case "log.modules":
		return "*"
	case "log.rotate":
		return "false"
	case "log.max_size":
		return "10m"
	case "log.max_files":
		return "5"
	case "vm.cpus":
		return "4"
	case "vm.memory":
		return "8192"
	case "node_version":
		return "22"
	case "persistent":
		return "false"
	case "ports.forward":
		return "true"
	case "ports.expose":
		return ""
	case "ports.inject_system_prompt":
		return "true"
	case "ports.range_start":
		return "30000"
	case "ssh.forward_keys":
		return "true"
	case "ssh.forward_mode":
		return "proxy"
	case "ssh.allowed_keys":
		return ""
	case "history_persist":
		return "false"
	case "uv_version":
		return "latest"
	case "workdir.path":
		return "."
	case "workdir.automount":
		return "true"
	case "workdir.readonly":
		return "false"
	// Security defaults
	case "security.cap_add":
		return "CHOWN,SETUID,SETGID"
	case "security.cap_drop":
		return "ALL"
	case "security.disable_devices":
		return "false"
	case "security.disable_ipc":
		return "false"
	case "security.memory_swap":
		return ""
	case "security.network_mode":
		return ""
	case "security.no_new_privileges":
		return "true"
	case "security.pids_limit":
		return "200"
	case "security.read_only_rootfs":
		return "false"
	case "security.seccomp_profile":
		return ""
	case "security.isolate_secrets":
		return "false"
	case "security.time_limit":
		return "0"
	case "security.tmpfs_home_size":
		return "512m"
	case "security.tmpfs_tmp_size":
		return "256m"
	case "security.ulimit_nofile":
		return "4096:8192"
	case "security.ulimit_nproc":
		return "256:512"
	case "security.user_namespace":
		return ""
	// OTEL defaults
	case "otel.enabled":
		return "false"
	case "otel.endpoint":
		return "http://host.docker.internal:4318"
	case "otel.protocol":
		return "http/json"
	case "otel.service_name":
		return "addt"
	case "otel.headers":
		return ""
	}
	return ""
}

// IsValidKey checks if a key is a valid config key
func IsValidKey(key string) bool {
	for _, k := range GetKeys() {
		if k.Key == key {
			return true
		}
	}
	return false
}

// GetKeyInfo returns the metadata for a config key, or nil if not found
func GetKeyInfo(key string) *KeyInfo {
	for _, k := range GetKeys() {
		if k.Key == key {
			return &k
		}
	}
	return nil
}

// GetExtensionFlagKeys returns dynamic extension keys derived from an extension's config.yaml flags
func GetExtensionFlagKeys(extName string) []KeyInfo {
	exts, err := extensions.GetExtensions()
	if err != nil {
		return nil
	}
	var keys []KeyInfo
	for _, ext := range exts {
		if ext.Name != extName {
			continue
		}
		for _, flag := range ext.Flags {
			if flag.EnvVar == "" {
				continue
			}
			// Strip leading "--" from the flag name to get the key
			key := strings.TrimPrefix(flag.Flag, "--")
			keys = append(keys, KeyInfo{
				Key:         key,
				Description: flag.Description,
				Type:        "bool",
				EnvVar:      flag.EnvVar,
			})
		}
		break
	}
	return keys
}

// GetAllExtensionKeys returns both static and dynamic (flag) keys for an extension
func GetAllExtensionKeys(extName string) []KeyInfo {
	keys := GetExtensionKeys()
	keys = append(keys, GetExtensionFlagKeys(extName)...)
	return keys
}

// AvailableExtensionKeyNames returns a comma-separated list of all valid extension key names
func AvailableExtensionKeyNames(extName string) string {
	keys := GetAllExtensionKeys(extName)
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = k.Key
	}
	return strings.Join(names, ", ")
}

// IsValidExtensionKey checks if a key is a valid extension config key (static or dynamic flag)
func IsValidExtensionKey(key string, extName string) bool {
	for _, k := range GetAllExtensionKeys(extName) {
		if k.Key == key {
			return true
		}
	}
	return false
}

// IsFlagKey checks if a key corresponds to a dynamic flag key for the given extension
func IsFlagKey(key string, extName string) bool {
	for _, k := range GetExtensionFlagKeys(extName) {
		if k.Key == key {
			return true
		}
	}
	return false
}

// GetValue retrieves a config value from the config struct
func GetValue(cfg *cfgtypes.GlobalConfig, key string) string {
	switch key {
	case "env_file_load":
		if cfg.EnvFileLoad != nil {
			return fmt.Sprintf("%v", *cfg.EnvFileLoad)
		}
	case "env_file":
		return cfg.EnvFile
	case "go_version":
		return cfg.GoVersion
	case "node_version":
		return cfg.NodeVersion
	case "persistent":
		if cfg.Persistent != nil {
			return fmt.Sprintf("%v", *cfg.Persistent)
		}
	case "history_persist":
		if cfg.HistoryPersist != nil {
			return fmt.Sprintf("%v", *cfg.HistoryPersist)
		}
	case "uv_version":
		return cfg.UvVersion
	}
	// Check workdir keys
	if strings.HasPrefix(key, "workdir.") {
		return GetWorkdirValue(cfg.Workdir, key)
	}
	// Check firewall keys
	if strings.HasPrefix(key, "firewall.") {
		return GetFirewallValue(cfg.Firewall, key)
	}
	// Check github keys
	if strings.HasPrefix(key, "github.") {
		return GetGitHubValue(cfg.GitHub, key)
	}
	// Check GPG keys
	if strings.HasPrefix(key, "gpg.") {
		return GetGPGValue(cfg.GPG, key)
	}
	// Check log keys
	if strings.HasPrefix(key, "log.") {
		return GetLogValue(cfg.Log, key)
	}
	// Check ports keys
	if strings.HasPrefix(key, "ports.") {
		return GetPortsValue(cfg.Ports, key)
	}
	// Check SSH keys
	if strings.HasPrefix(key, "ssh.") {
		return GetSSHValue(cfg.SSH, key)
	}
	// Check container keys
	if strings.HasPrefix(key, "container.") {
		return GetContainerValue(cfg.Container, key)
	}
	// Check VM keys
	if strings.HasPrefix(key, "vm.") {
		return GetVmValue(cfg.Vm, key)
	}
	// Check docker keys
	if strings.HasPrefix(key, "docker.") {
		return GetDockerValue(cfg.Docker, key)
	}
	// Check security keys
	if strings.HasPrefix(key, "security.") {
		return GetSecurityValue(cfg.Security, key)
	}
	// Check OTEL keys
	if strings.HasPrefix(key, "otel.") {
		return GetOtelValue(cfg.Otel, key)
	}
	return ""
}

// SetValue sets a config value in the config struct
func SetValue(cfg *cfgtypes.GlobalConfig, key, value string) {
	switch key {
	case "env_file_load":
		b := value == "true"
		cfg.EnvFileLoad = &b
	case "env_file":
		cfg.EnvFile = value
	case "go_version":
		cfg.GoVersion = value
	case "node_version":
		cfg.NodeVersion = value
	case "persistent":
		b := value == "true"
		cfg.Persistent = &b
	case "history_persist":
		b := value == "true"
		cfg.HistoryPersist = &b
	case "uv_version":
		cfg.UvVersion = value
	default:
		// Check workdir keys
		if strings.HasPrefix(key, "workdir.") {
			if cfg.Workdir == nil {
				cfg.Workdir = &cfgtypes.WorkdirSettings{}
			}
			SetWorkdirValue(cfg.Workdir, key, value)
		}
		// Check firewall keys
		if strings.HasPrefix(key, "firewall.") {
			if cfg.Firewall == nil {
				cfg.Firewall = &cfgtypes.FirewallSettings{}
			}
			SetFirewallValue(cfg.Firewall, key, value)
		}
		// Check github keys
		if strings.HasPrefix(key, "github.") {
			if cfg.GitHub == nil {
				cfg.GitHub = &cfgtypes.GitHubSettings{}
			}
			SetGitHubValue(cfg.GitHub, key, value)
		}
		// Check GPG keys
		if strings.HasPrefix(key, "gpg.") {
			if cfg.GPG == nil {
				cfg.GPG = &cfgtypes.GPGSettings{}
			}
			SetGPGValue(cfg.GPG, key, value)
		}
		// Check log keys
		if strings.HasPrefix(key, "log.") {
			if cfg.Log == nil {
				cfg.Log = &cfgtypes.LogSettings{}
			}
			SetLogValue(cfg.Log, key, value)
		}
		// Check ports keys
		if strings.HasPrefix(key, "ports.") {
			if cfg.Ports == nil {
				cfg.Ports = &cfgtypes.PortsSettings{}
			}
			SetPortsValue(cfg.Ports, key, value)
		}
		// Check SSH keys
		if strings.HasPrefix(key, "ssh.") {
			if cfg.SSH == nil {
				cfg.SSH = &cfgtypes.SSHSettings{}
			}
			SetSSHValue(cfg.SSH, key, value)
		}
		// Check container keys
		if strings.HasPrefix(key, "container.") {
			if cfg.Container == nil {
				cfg.Container = &cfgtypes.ContainerSettings{}
			}
			SetContainerValue(cfg.Container, key, value)
		}
		// Check VM keys
		if strings.HasPrefix(key, "vm.") {
			if cfg.Vm == nil {
				cfg.Vm = &cfgtypes.VmSettings{}
			}
			SetVmValue(cfg.Vm, key, value)
		}
		// Check docker keys
		if strings.HasPrefix(key, "docker.") {
			if cfg.Docker == nil {
				cfg.Docker = &cfgtypes.DockerSettings{}
			}
			SetDockerValue(cfg.Docker, key, value)
		}
		// Check security keys
		if strings.HasPrefix(key, "security.") {
			if cfg.Security == nil {
				cfg.Security = &security.Settings{}
			}
			SetSecurityValue(cfg.Security, key, value)
		}
		// Check OTEL keys
		if strings.HasPrefix(key, "otel.") {
			if cfg.Otel == nil {
				cfg.Otel = &otel.Settings{}
			}
			SetOtelValue(cfg.Otel, key, value)
		}
	}
}

// UnsetValue clears a config value in the config struct
func UnsetValue(cfg *cfgtypes.GlobalConfig, key string) {
	switch key {
	case "env_file_load":
		cfg.EnvFileLoad = nil
	case "env_file":
		cfg.EnvFile = ""
	case "go_version":
		cfg.GoVersion = ""
	case "node_version":
		cfg.NodeVersion = ""
	case "persistent":
		cfg.Persistent = nil
	case "history_persist":
		cfg.HistoryPersist = nil
	case "uv_version":
		cfg.UvVersion = ""
	default:
		// Check workdir keys
		if strings.HasPrefix(key, "workdir.") && cfg.Workdir != nil {
			UnsetWorkdirValue(cfg.Workdir, key)
		}
		// Check firewall keys
		if strings.HasPrefix(key, "firewall.") && cfg.Firewall != nil {
			UnsetFirewallValue(cfg.Firewall, key)
		}
		// Check github keys
		if strings.HasPrefix(key, "github.") && cfg.GitHub != nil {
			UnsetGitHubValue(cfg.GitHub, key)
		}
		// Check GPG keys
		if strings.HasPrefix(key, "gpg.") && cfg.GPG != nil {
			UnsetGPGValue(cfg.GPG, key)
		}
		// Check log keys
		if strings.HasPrefix(key, "log.") && cfg.Log != nil {
			UnsetLogValue(cfg.Log, key)
		}
		// Check ports keys
		if strings.HasPrefix(key, "ports.") && cfg.Ports != nil {
			UnsetPortsValue(cfg.Ports, key)
		}
		// Check SSH keys
		if strings.HasPrefix(key, "ssh.") && cfg.SSH != nil {
			UnsetSSHValue(cfg.SSH, key)
		}
		// Check container keys
		if strings.HasPrefix(key, "container.") && cfg.Container != nil {
			UnsetContainerValue(cfg.Container, key)
		}
		// Check VM keys
		if strings.HasPrefix(key, "vm.") && cfg.Vm != nil {
			UnsetVmValue(cfg.Vm, key)
		}
		// Check docker keys
		if strings.HasPrefix(key, "docker.") && cfg.Docker != nil {
			UnsetDockerValue(cfg.Docker, key)
		}
		// Check security keys
		if strings.HasPrefix(key, "security.") && cfg.Security != nil {
			UnsetSecurityValue(cfg.Security, key)
		}
		// Check OTEL keys
		if strings.HasPrefix(key, "otel.") && cfg.Otel != nil {
			UnsetOtelValue(cfg.Otel, key)
		}
	}
}
