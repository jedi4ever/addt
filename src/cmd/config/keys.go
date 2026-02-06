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
		{Key: "firewall", Description: "Enable network firewall", Type: "bool", EnvVar: "ADDT_FIREWALL"},
		{Key: "firewall_mode", Description: "Firewall mode: strict, permissive, off", Type: "string", EnvVar: "ADDT_FIREWALL_MODE"},
		{Key: "github_detect", Description: "Auto-detect GitHub token from gh CLI", Type: "bool", EnvVar: "ADDT_GITHUB_DETECT"},
		{Key: "go_version", Description: "Go version", Type: "string", EnvVar: "ADDT_GO_VERSION"},
		{Key: "gpg_forward", Description: "Enable GPG forwarding", Type: "bool", EnvVar: "ADDT_GPG_FORWARD"},
		{Key: "log", Description: "Enable command logging", Type: "bool", EnvVar: "ADDT_LOG"},
		{Key: "log_file", Description: "Log file path", Type: "string", EnvVar: "ADDT_LOG_FILE"},
		{Key: "node_version", Description: "Node.js version", Type: "string", EnvVar: "ADDT_NODE_VERSION"},
		{Key: "persistent", Description: "Enable persistent container mode", Type: "bool", EnvVar: "ADDT_PERSISTENT"},
		{Key: "port_range_start", Description: "Starting port for auto allocation", Type: "int", EnvVar: "ADDT_PORT_RANGE_START"},
		{Key: "ssh_forward", Description: "SSH forwarding mode: proxy, agent, or keys (default: proxy)", Type: "string", EnvVar: "ADDT_SSH_FORWARD"},
		{Key: "ssh_allowed_keys", Description: "Key filters for proxy mode (comma-separated)", Type: "string", EnvVar: "ADDT_SSH_ALLOWED_KEYS"},
		{Key: "history_persist", Description: "Persist shell history between sessions (default: false)", Type: "bool", EnvVar: "ADDT_HISTORY_PERSIST"},
		{Key: "uv_version", Description: "UV Python package manager version", Type: "string", EnvVar: "ADDT_UV_VERSION"},
		{Key: "workdir", Description: "Override working directory (default: current directory)", Type: "string", EnvVar: "ADDT_WORKDIR"},
		{Key: "workdir_automount", Description: "Auto-mount working directory to /workspace", Type: "bool", EnvVar: "ADDT_WORKDIR_AUTOMOUNT"},
	}
	// Add docker keys
	keys = append(keys, GetDockerKeys()...)
	// Add security keys
	keys = append(keys, GetSecurityKeys()...)
	// Add OTEL keys
	keys = append(keys, GetOtelKeys()...)
	return keys
}

// GetDockerKeys returns all valid Docker config keys
func GetDockerKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "docker.cpus", Description: "CPU limit for container (e.g., \"2\", \"0.5\")", Type: "string", EnvVar: "ADDT_DOCKER_CPUS"},
		{Key: "docker.dind.enable", Description: "Enable Docker-in-Docker", Type: "bool", EnvVar: "ADDT_DOCKER_DIND_ENABLE"},
		{Key: "docker.dind.mode", Description: "Docker-in-Docker mode: host or isolated", Type: "string", EnvVar: "ADDT_DOCKER_DIND_MODE"},
		{Key: "docker.memory", Description: "Memory limit for container (e.g., \"512m\", \"2g\")", Type: "string", EnvVar: "ADDT_DOCKER_MEMORY"},
	}
}

// GetSecurityKeys returns all valid security config keys
func GetSecurityKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "security.cap_add", Description: "Capabilities to add (comma-separated)", Type: "string", EnvVar: "ADDT_SECURITY_CAP_ADD"},
		{Key: "security.cap_drop", Description: "Capabilities to drop (comma-separated)", Type: "string", EnvVar: "ADDT_SECURITY_CAP_DROP"},
		{Key: "security.disable_devices", Description: "Drop MKNOD capability", Type: "bool", EnvVar: "ADDT_SECURITY_DISABLE_DEVICES"},
		{Key: "security.disable_ipc", Description: "Disable IPC namespace sharing", Type: "bool", EnvVar: "ADDT_SECURITY_DISABLE_IPC"},
		{Key: "security.memory_swap", Description: "Memory swap limit (\"-1\" to disable)", Type: "string", EnvVar: "ADDT_SECURITY_MEMORY_SWAP"},
		{Key: "security.network_mode", Description: "Network mode: bridge, none, host", Type: "string", EnvVar: "ADDT_SECURITY_NETWORK_MODE"},
		{Key: "security.no_new_privileges", Description: "Prevent privilege escalation", Type: "bool", EnvVar: "ADDT_SECURITY_NO_NEW_PRIVILEGES"},
		{Key: "security.pids_limit", Description: "Max number of processes", Type: "int", EnvVar: "ADDT_SECURITY_PIDS_LIMIT"},
		{Key: "security.read_only_rootfs", Description: "Read-only root filesystem", Type: "bool", EnvVar: "ADDT_SECURITY_READ_ONLY_ROOTFS"},
		{Key: "security.seccomp_profile", Description: "Seccomp profile: default, restrictive, unconfined", Type: "string", EnvVar: "ADDT_SECURITY_SECCOMP_PROFILE"},
		{Key: "security.isolate_secrets", Description: "Isolate secrets from child processes", Type: "bool", EnvVar: "ADDT_SECURITY_ISOLATE_SECRETS"},
		{Key: "security.time_limit", Description: "Auto-kill after N minutes (0=disabled)", Type: "int", EnvVar: "ADDT_SECURITY_TIME_LIMIT"},
		{Key: "security.tmpfs_home_size", Description: "Size of /home tmpfs (e.g., \"512m\")", Type: "string", EnvVar: "ADDT_SECURITY_TMPFS_HOME_SIZE"},
		{Key: "security.tmpfs_tmp_size", Description: "Size of /tmp tmpfs (e.g., \"256m\")", Type: "string", EnvVar: "ADDT_SECURITY_TMPFS_TMP_SIZE"},
		{Key: "security.ulimit_nofile", Description: "File descriptor limit (soft:hard)", Type: "string", EnvVar: "ADDT_SECURITY_ULIMIT_NOFILE"},
		{Key: "security.ulimit_nproc", Description: "Process limit (soft:hard)", Type: "string", EnvVar: "ADDT_SECURITY_ULIMIT_NPROC"},
		{Key: "security.user_namespace", Description: "User namespace: host, private", Type: "string", EnvVar: "ADDT_SECURITY_USER_NAMESPACE"},
	}
}

// GetOtelKeys returns all valid OpenTelemetry config keys
func GetOtelKeys() []KeyInfo {
	return []KeyInfo{
		{Key: "otel.enabled", Description: "Enable OpenTelemetry", Type: "bool", EnvVar: "ADDT_OTEL_ENABLED"},
		{Key: "otel.endpoint", Description: "OTLP endpoint URL", Type: "string", EnvVar: "ADDT_OTEL_ENDPOINT"},
		{Key: "otel.protocol", Description: "OTLP protocol: http/json, http/protobuf, or grpc", Type: "string", EnvVar: "ADDT_OTEL_PROTOCOL"},
		{Key: "otel.service_name", Description: "Service name for telemetry", Type: "string", EnvVar: "ADDT_OTEL_SERVICE_NAME"},
		{Key: "otel.headers", Description: "OTLP headers (key=value,key2=value2)", Type: "string", EnvVar: "ADDT_OTEL_HEADERS"},
	}
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
	case "docker.cpus":
		return ""
	case "docker.dind.enable":
		return "false"
	case "docker.dind.mode":
		return "isolated"
	case "firewall":
		return "false"
	case "firewall_mode":
		return "strict"
	case "github_detect":
		return "false"
	case "go_version":
		return "latest"
	case "gpg_forward":
		return "false"
	case "log":
		return "false"
	case "log_file":
		return "addt.log"
	case "docker.memory":
		return ""
	case "node_version":
		return "22"
	case "persistent":
		return "false"
	case "port_range_start":
		return "30000"
	case "ssh_forward":
		return "agent"
	case "history_persist":
		return "false"
	case "uv_version":
		return "latest"
	case "workdir":
		return "."
	case "workdir_automount":
		return "true"
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
	case "firewall":
		if cfg.Firewall != nil {
			return fmt.Sprintf("%v", *cfg.Firewall)
		}
	case "firewall_mode":
		return cfg.FirewallMode
	case "github_detect":
		if cfg.GitHubDetect != nil {
			return fmt.Sprintf("%v", *cfg.GitHubDetect)
		}
	case "go_version":
		return cfg.GoVersion
	case "gpg_forward":
		return cfg.GPGForward
	case "gpg_allowed_key_ids":
		return strings.Join(cfg.GPGAllowedKeyIDs, ",")
	case "log":
		if cfg.Log != nil {
			return fmt.Sprintf("%v", *cfg.Log)
		}
	case "log_file":
		return cfg.LogFile
	case "node_version":
		return cfg.NodeVersion
	case "persistent":
		if cfg.Persistent != nil {
			return fmt.Sprintf("%v", *cfg.Persistent)
		}
	case "port_range_start":
		if cfg.PortRangeStart != nil {
			return fmt.Sprintf("%d", *cfg.PortRangeStart)
		}
	case "ssh_forward":
		return cfg.SSHForward
	case "history_persist":
		if cfg.HistoryPersist != nil {
			return fmt.Sprintf("%v", *cfg.HistoryPersist)
		}
	case "uv_version":
		return cfg.UvVersion
	case "workdir":
		return cfg.Workdir
	case "workdir_automount":
		if cfg.WorkdirAutomount != nil {
			return fmt.Sprintf("%v", *cfg.WorkdirAutomount)
		}
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

// GetSecurityValue retrieves a security config value
func GetSecurityValue(sec *security.Settings, key string) string {
	if sec == nil {
		return ""
	}
	switch key {
	case "security.cap_add":
		return strings.Join(sec.CapAdd, ",")
	case "security.cap_drop":
		return strings.Join(sec.CapDrop, ",")
	case "security.disable_devices":
		if sec.DisableDevices != nil {
			return fmt.Sprintf("%v", *sec.DisableDevices)
		}
	case "security.disable_ipc":
		if sec.DisableIPC != nil {
			return fmt.Sprintf("%v", *sec.DisableIPC)
		}
	case "security.memory_swap":
		return sec.MemorySwap
	case "security.network_mode":
		return sec.NetworkMode
	case "security.no_new_privileges":
		if sec.NoNewPrivileges != nil {
			return fmt.Sprintf("%v", *sec.NoNewPrivileges)
		}
	case "security.pids_limit":
		if sec.PidsLimit != nil {
			return fmt.Sprintf("%d", *sec.PidsLimit)
		}
	case "security.read_only_rootfs":
		if sec.ReadOnlyRootfs != nil {
			return fmt.Sprintf("%v", *sec.ReadOnlyRootfs)
		}
	case "security.seccomp_profile":
		return sec.SeccompProfile
	case "security.isolate_secrets":
		if sec.IsolateSecrets != nil {
			return fmt.Sprintf("%v", *sec.IsolateSecrets)
		}
	case "security.time_limit":
		if sec.TimeLimit > 0 {
			return fmt.Sprintf("%d", sec.TimeLimit)
		}
		return "0"
	case "security.tmpfs_home_size":
		return sec.TmpfsHomeSize
	case "security.tmpfs_tmp_size":
		return sec.TmpfsTmpSize
	case "security.ulimit_nofile":
		return sec.UlimitNofile
	case "security.ulimit_nproc":
		return sec.UlimitNproc
	case "security.user_namespace":
		return sec.UserNamespace
	}
	return ""
}

// GetOtelValue retrieves an OTEL config value
func GetOtelValue(o *otel.Settings, key string) string {
	if o == nil {
		return ""
	}
	switch key {
	case "otel.enabled":
		if o.Enabled != nil {
			return fmt.Sprintf("%v", *o.Enabled)
		}
	case "otel.endpoint":
		if o.Endpoint != nil {
			return *o.Endpoint
		}
	case "otel.protocol":
		if o.Protocol != nil {
			return *o.Protocol
		}
	case "otel.service_name":
		if o.ServiceName != nil {
			return *o.ServiceName
		}
	case "otel.headers":
		if o.Headers != nil {
			return *o.Headers
		}
	}
	return ""
}

// GetDockerValue retrieves a Docker config value
func GetDockerValue(d *cfgtypes.DockerSettings, key string) string {
	if d == nil {
		return ""
	}
	switch key {
	case "docker.cpus":
		return d.CPUs
	case "docker.memory":
		return d.Memory
	case "docker.dind.enable":
		if d.Dind != nil && d.Dind.Enable != nil {
			return fmt.Sprintf("%v", *d.Dind.Enable)
		}
	case "docker.dind.mode":
		if d.Dind != nil {
			return d.Dind.Mode
		}
	}
	return ""
}

// SetValue sets a config value in the config struct
func SetValue(cfg *cfgtypes.GlobalConfig, key, value string) {
	switch key {
	case "firewall":
		b := value == "true"
		cfg.Firewall = &b
	case "firewall_mode":
		cfg.FirewallMode = value
	case "github_detect":
		b := value == "true"
		cfg.GitHubDetect = &b
	case "go_version":
		cfg.GoVersion = value
	case "gpg_forward":
		cfg.GPGForward = value
	case "gpg_allowed_key_ids":
		if value == "" {
			cfg.GPGAllowedKeyIDs = nil
		} else {
			cfg.GPGAllowedKeyIDs = strings.Split(value, ",")
		}
	case "log":
		b := value == "true"
		cfg.Log = &b
	case "log_file":
		cfg.LogFile = value
	case "node_version":
		cfg.NodeVersion = value
	case "persistent":
		b := value == "true"
		cfg.Persistent = &b
	case "port_range_start":
		var i int
		fmt.Sscanf(value, "%d", &i)
		cfg.PortRangeStart = &i
	case "ssh_forward":
		cfg.SSHForward = value
	case "history_persist":
		b := value == "true"
		cfg.HistoryPersist = &b
	case "uv_version":
		cfg.UvVersion = value
	case "workdir":
		cfg.Workdir = value
	case "workdir_automount":
		b := value == "true"
		cfg.WorkdirAutomount = &b
	default:
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

// SetSecurityValue sets a security config value
func SetSecurityValue(sec *security.Settings, key, value string) {
	switch key {
	case "security.cap_add":
		if value == "" {
			sec.CapAdd = nil
		} else {
			sec.CapAdd = strings.Split(value, ",")
		}
	case "security.cap_drop":
		if value == "" {
			sec.CapDrop = nil
		} else {
			sec.CapDrop = strings.Split(value, ",")
		}
	case "security.disable_devices":
		b := value == "true"
		sec.DisableDevices = &b
	case "security.disable_ipc":
		b := value == "true"
		sec.DisableIPC = &b
	case "security.memory_swap":
		sec.MemorySwap = value
	case "security.network_mode":
		sec.NetworkMode = value
	case "security.no_new_privileges":
		b := value == "true"
		sec.NoNewPrivileges = &b
	case "security.pids_limit":
		var i int
		fmt.Sscanf(value, "%d", &i)
		sec.PidsLimit = &i
	case "security.read_only_rootfs":
		b := value == "true"
		sec.ReadOnlyRootfs = &b
	case "security.seccomp_profile":
		sec.SeccompProfile = value
	case "security.isolate_secrets":
		b := value == "true"
		sec.IsolateSecrets = &b
	case "security.time_limit":
		var i int
		fmt.Sscanf(value, "%d", &i)
		sec.TimeLimit = i
	case "security.tmpfs_home_size":
		sec.TmpfsHomeSize = value
	case "security.tmpfs_tmp_size":
		sec.TmpfsTmpSize = value
	case "security.ulimit_nofile":
		sec.UlimitNofile = value
	case "security.ulimit_nproc":
		sec.UlimitNproc = value
	case "security.user_namespace":
		sec.UserNamespace = value
	}
}

// SetOtelValue sets an OTEL config value
func SetOtelValue(o *otel.Settings, key, value string) {
	switch key {
	case "otel.enabled":
		b := value == "true"
		o.Enabled = &b
	case "otel.endpoint":
		o.Endpoint = &value
	case "otel.protocol":
		o.Protocol = &value
	case "otel.service_name":
		o.ServiceName = &value
	case "otel.headers":
		o.Headers = &value
	}
}

// SetDockerValue sets a Docker config value
func SetDockerValue(d *cfgtypes.DockerSettings, key, value string) {
	switch key {
	case "docker.cpus":
		d.CPUs = value
	case "docker.memory":
		d.Memory = value
	case "docker.dind.enable":
		if d.Dind == nil {
			d.Dind = &cfgtypes.DindSettings{}
		}
		b := value == "true"
		d.Dind.Enable = &b
	case "docker.dind.mode":
		if d.Dind == nil {
			d.Dind = &cfgtypes.DindSettings{}
		}
		d.Dind.Mode = value
	}
}

// UnsetValue clears a config value in the config struct
func UnsetValue(cfg *cfgtypes.GlobalConfig, key string) {
	switch key {
	case "firewall":
		cfg.Firewall = nil
	case "firewall_mode":
		cfg.FirewallMode = ""
	case "github_detect":
		cfg.GitHubDetect = nil
	case "go_version":
		cfg.GoVersion = ""
	case "gpg_forward":
		cfg.GPGForward = ""
	case "gpg_allowed_key_ids":
		cfg.GPGAllowedKeyIDs = nil
	case "log":
		cfg.Log = nil
	case "log_file":
		cfg.LogFile = ""
	case "node_version":
		cfg.NodeVersion = ""
	case "persistent":
		cfg.Persistent = nil
	case "port_range_start":
		cfg.PortRangeStart = nil
	case "ssh_forward":
		cfg.SSHForward = ""
	case "history_persist":
		cfg.HistoryPersist = nil
	case "uv_version":
		cfg.UvVersion = ""
	case "workdir":
		cfg.Workdir = ""
	case "workdir_automount":
		cfg.WorkdirAutomount = nil
	default:
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

// UnsetSecurityValue clears a security config value
func UnsetSecurityValue(sec *security.Settings, key string) {
	switch key {
	case "security.cap_add":
		sec.CapAdd = nil
	case "security.cap_drop":
		sec.CapDrop = nil
	case "security.disable_devices":
		sec.DisableDevices = nil
	case "security.disable_ipc":
		sec.DisableIPC = nil
	case "security.memory_swap":
		sec.MemorySwap = ""
	case "security.network_mode":
		sec.NetworkMode = ""
	case "security.no_new_privileges":
		sec.NoNewPrivileges = nil
	case "security.pids_limit":
		sec.PidsLimit = nil
	case "security.read_only_rootfs":
		sec.ReadOnlyRootfs = nil
	case "security.seccomp_profile":
		sec.SeccompProfile = ""
	case "security.isolate_secrets":
		sec.IsolateSecrets = nil
	case "security.time_limit":
		sec.TimeLimit = 0
	case "security.tmpfs_home_size":
		sec.TmpfsHomeSize = ""
	case "security.tmpfs_tmp_size":
		sec.TmpfsTmpSize = ""
	case "security.ulimit_nofile":
		sec.UlimitNofile = ""
	case "security.ulimit_nproc":
		sec.UlimitNproc = ""
	case "security.user_namespace":
		sec.UserNamespace = ""
	}
}

// UnsetOtelValue clears an OTEL config value
func UnsetOtelValue(o *otel.Settings, key string) {
	switch key {
	case "otel.enabled":
		o.Enabled = nil
	case "otel.endpoint":
		o.Endpoint = nil
	case "otel.protocol":
		o.Protocol = nil
	case "otel.service_name":
		o.ServiceName = nil
	case "otel.headers":
		o.Headers = nil
	}
}

// UnsetDockerValue clears a Docker config value
func UnsetDockerValue(d *cfgtypes.DockerSettings, key string) {
	switch key {
	case "docker.cpus":
		d.CPUs = ""
	case "docker.memory":
		d.Memory = ""
	case "docker.dind.enable":
		if d.Dind != nil {
			d.Dind.Enable = nil
		}
	case "docker.dind.mode":
		if d.Dind != nil {
			d.Dind.Mode = ""
		}
	}
}
