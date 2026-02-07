package config

import (
	"fmt"
	"strings"

	"github.com/jedi4ever/addt/config/security"
)

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
		if sec.TimeLimit != nil {
			return fmt.Sprintf("%d", *sec.TimeLimit)
		}
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
		sec.TimeLimit = &i
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
		sec.TimeLimit = nil
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
